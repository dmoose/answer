/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package fastgate_connector

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// pendingTTL bounds how long a login attempt may sit between the
	// /authorize redirect and the callback before it is rejected.
	pendingTTL = 10 * time.Minute
	// jwksTTL is how long fetched signing keys are trusted before refetch.
	jwksTTL = time.Hour
	// clockSkew tolerated when checking exp/iat.
	clockSkew = 2 * time.Minute
	// bindCookieName carries the browser-binding secret so a callback can
	// only complete in the browser that started the flow.
	bindCookieName = "fg_oidc_bind"
)

// pendingAuth is the server-side record of one in-flight login attempt,
// keyed by the opaque state value. Single-use: consumed on callback.
type pendingAuth struct {
	nonce    string
	verifier string
	bind     string
	expires  time.Time
}

type pendingStore struct {
	mu      sync.Mutex
	entries map[string]pendingAuth
}

func newPendingStore() *pendingStore {
	return &pendingStore{entries: map[string]pendingAuth{}}
}

func (s *pendingStore) put(state string, p pendingAuth) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, v := range s.entries {
		if now.After(v.expires) {
			delete(s.entries, k)
		}
	}
	s.entries[state] = p
}

// take returns and removes the entry for state; a second call with the same
// state fails, which is what rejects replayed callbacks.
func (s *pendingStore) take(state string) (pendingAuth, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.entries[state]
	if !ok {
		return pendingAuth{}, false
	}
	delete(s.entries, state)
	if time.Now().After(p.expires) {
		return pendingAuth{}, false
	}
	return p, true
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func pkceChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// jwksCache fetches and caches the issuer's Ed25519 signing keys by kid.
type jwksCache struct {
	mu        sync.Mutex
	keys      map[string]ed25519.PublicKey
	fetchedAt time.Time
}

func (j *jwksCache) key(client *http.Client, issuer, kid string) (ed25519.PublicKey, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if k, ok := j.keys[kid]; ok && time.Since(j.fetchedAt) < jwksTTL {
		return k, nil
	}
	resp, err := client.Get(strings.TrimRight(issuer, "/") + "/jwks")
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch jwks: status %d", resp.StatusCode)
	}
	var doc struct {
		Keys []struct {
			Kty string `json:"kty"`
			Crv string `json:"crv"`
			Kid string `json:"kid"`
			X   string `json:"x"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}
	keys := map[string]ed25519.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kty != "OKP" || k.Crv != "Ed25519" {
			continue
		}
		raw, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			continue
		}
		keys[k.Kid] = ed25519.PublicKey(raw)
	}
	j.keys = keys
	j.fetchedAt = time.Now()
	k, ok := keys[kid]
	if !ok {
		return nil, fmt.Errorf("jwks has no key %q", kid)
	}
	return k, nil
}

// idTokenClaims are the claims we require from a verified ID token.
type idTokenClaims struct {
	Iss           string        `json:"iss"`
	Sub           string        `json:"sub"`
	Aud           audienceClaim `json:"aud"`
	Exp           int64         `json:"exp"`
	Iat           int64         `json:"iat"`
	Nonce         string        `json:"nonce"`
	Email         string        `json:"email"`
	EmailVerified bool          `json:"email_verified"`
}

// audienceClaim accepts both the string and array forms RFC 7519 allows.
type audienceClaim []string

func (a *audienceClaim) UnmarshalJSON(b []byte) error {
	var one string
	if err := json.Unmarshal(b, &one); err == nil {
		*a = []string{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return err
	}
	*a = many
	return nil
}

func (a audienceClaim) contains(aud string) bool {
	return slices.Contains(a, aud)
}

// verifyIDToken checks the compact JWT's EdDSA signature against the
// issuer's JWKS and validates iss, aud, exp, iat and nonce.
func verifyIDToken(client *http.Client, jwks *jwksCache, issuer, clientID, token, expectedNonce string) (*idTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("id_token is not a compact JWT")
	}
	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode id_token header: %w", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerRaw, &header); err != nil {
		return nil, fmt.Errorf("parse id_token header: %w", err)
	}
	if header.Alg != "EdDSA" {
		return nil, fmt.Errorf("unsupported id_token alg %q", header.Alg)
	}
	pub, err := jwks.key(client, issuer, header.Kid)
	if err != nil {
		return nil, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode id_token signature: %w", err)
	}
	if !ed25519.Verify(pub, []byte(parts[0]+"."+parts[1]), sig) {
		return nil, fmt.Errorf("id_token signature invalid")
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode id_token payload: %w", err)
	}
	claims := &idTokenClaims{}
	if err := json.Unmarshal(payloadRaw, claims); err != nil {
		return nil, fmt.Errorf("parse id_token claims: %w", err)
	}
	if claims.Sub == "" {
		return nil, fmt.Errorf("id_token has no subject")
	}
	now := time.Now()
	if strings.TrimRight(claims.Iss, "/") != strings.TrimRight(issuer, "/") {
		return nil, fmt.Errorf("id_token issuer %q does not match %q", claims.Iss, issuer)
	}
	if !claims.Aud.contains(clientID) {
		return nil, fmt.Errorf("id_token audience %v does not include client", claims.Aud)
	}
	if claims.Exp <= 0 || now.After(time.Unix(claims.Exp, 0).Add(clockSkew)) {
		return nil, fmt.Errorf("id_token expired")
	}
	if claims.Iat > 0 && time.Unix(claims.Iat, 0).After(now.Add(clockSkew)) {
		return nil, fmt.Errorf("id_token issued in the future")
	}
	if expectedNonce == "" || subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(expectedNonce)) != 1 {
		return nil, fmt.Errorf("id_token nonce mismatch")
	}
	return claims, nil
}
