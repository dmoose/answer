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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testClientID = "test-client"

// stubIdP is a minimal fastgate stand-in: EdDSA-signed ID tokens, /jwks,
// /token (PKCE-checking) and /userinfo.
type stubIdP struct {
	t          *testing.T
	pub        ed25519.PublicKey
	priv       ed25519.PrivateKey
	kid        string
	server     *httptest.Server
	issuer     string
	expectedCV string // code_verifier the /token endpoint requires, "" to skip

	// knobs for failure-injection
	idTokenOverride  func(claims map[string]any) // mutate claims pre-sign
	tamperSignature  bool
	userinfoStatus   int
	emailVerified    bool
	userinfoSubject  string
	omitIDToken      bool
	lastTokenRequest url.Values
}

func newStubIdP(t *testing.T) *stubIdP {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	s := &stubIdP{
		t: t, pub: pub, priv: priv, kid: "test-key",
		userinfoStatus: http.StatusOK, emailVerified: true,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{
			"kty": "OKP", "crv": "Ed25519", "use": "sig", "alg": "EdDSA",
			"kid": s.kid, "x": base64.RawURLEncoding.EncodeToString(s.pub),
		}}})
	})
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		s.lastTokenRequest = r.PostForm
		if s.expectedCV != "" && r.PostForm.Get("code_verifier") != s.expectedCV {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"error":"invalid_grant"}`)
			return
		}
		resp := map[string]any{"access_token": "stub-access-token", "token_type": "Bearer"}
		if !s.omitIDToken {
			resp["id_token"] = s.signIDToken()
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, r *http.Request) {
		if s.userinfoStatus != http.StatusOK {
			w.WriteHeader(s.userinfoStatus)
			return
		}
		sub := s.userinfoSubject
		if sub == "" {
			sub = "user-1"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub": sub, "email": "alice@example.com", "email_verified": s.emailVerified,
			"name": "Alice", "preferred_username": "alice", "picture": "https://cdn/p.png",
		})
	})
	s.server = httptest.NewServer(mux)
	t.Cleanup(s.server.Close)
	s.issuer = s.server.URL
	return s
}

// nonce is stored on the stub before /token is called via connectorFlow.
var stubNonce string

func (s *stubIdP) signIDToken() string {
	now := time.Now()
	claims := map[string]any{
		"iss": s.issuer, "sub": "user-1", "aud": testClientID,
		"exp": now.Add(10 * time.Minute).Unix(), "iat": now.Unix(),
		"nonce": stubNonce, "email": "alice@example.com", "email_verified": s.emailVerified,
	}
	if s.idTokenOverride != nil {
		s.idTokenOverride(claims)
	}
	header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "kid": s.kid, "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	signing := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	sig := ed25519.Sign(s.priv, []byte(signing))
	if s.tamperSignature {
		sig[0] ^= 0xFF
	}
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func newTestConnector(idp *stubIdP) *Connector {
	return &Connector{
		Config:  &ConnectorConfig{Issuer: idp.issuer, ClientID: testClientID, ClientSecret: "shhh"},
		pending: newPendingStore(),
		jwks:    &jwksCache{},
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func ginCtx(req *http.Request) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	return ctx, w
}

// startLogin drives ConnectorSender and returns the parsed authorize params
// plus the bind cookie the browser would hold.
func startLogin(t *testing.T, c *Connector) (params url.Values, bindCookie *http.Cookie) {
	req := httptest.NewRequest("GET", "http://answer.example/login", nil)
	ctx, w := ginCtx(req)
	redirect := c.ConnectorSender(ctx, "http://answer.example/receiver")
	require.NotEmpty(t, redirect)
	u, err := url.Parse(redirect)
	require.NoError(t, err)
	params = u.Query()
	for _, ck := range w.Result().Cookies() {
		if ck.Name == bindCookieName {
			bindCookie = ck
		}
	}
	require.NotNil(t, bindCookie, "sender must set the bind cookie")
	return params, bindCookie
}

// callback drives ConnectorReceiver with the given state and cookie.
func callback(c *Connector, state string, cookie *http.Cookie) (userInfo any, err error) {
	req := httptest.NewRequest("GET", "http://answer.example/receiver?code=abc&state="+url.QueryEscape(state), nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	ctx, _ := ginCtx(req)
	return c.ConnectorReceiver(ctx, "http://answer.example/receiver")
}

func TestSenderIssuesFreshStateNoncePKCE(t *testing.T) {
	idp := newStubIdP(t)
	c := newTestConnector(idp)

	p1, _ := startLogin(t, c)
	p2, _ := startLogin(t, c)

	assert.NotEmpty(t, p1.Get("state"))
	assert.NotEmpty(t, p1.Get("nonce"))
	assert.NotEmpty(t, p1.Get("code_challenge"))
	assert.Equal(t, "S256", p1.Get("code_challenge_method"))
	assert.NotEqual(t, p1.Get("state"), p2.Get("state"), "state must be per-request")
	assert.NotEqual(t, p1.Get("nonce"), p2.Get("nonce"), "nonce must be per-request")
	assert.NotEqual(t, p1.Get("code_challenge"), p2.Get("code_challenge"))
}

func TestReceiverRejectsMissingUnknownReplayedState(t *testing.T) {
	idp := newStubIdP(t)
	c := newTestConnector(idp)
	params, cookie := startLogin(t, c)
	stubNonce = params.Get("nonce")

	_, err := callback(c, "", cookie)
	assert.ErrorContains(t, err, "missing state")

	_, err = callback(c, "never-issued", cookie)
	assert.ErrorContains(t, err, "state")

	// Legitimate use consumes the state...
	_, err = callback(c, params.Get("state"), cookie)
	require.NoError(t, err)
	// ...and a replay of the same callback is rejected.
	_, err = callback(c, params.Get("state"), cookie)
	assert.ErrorContains(t, err, "replayed")
}

func TestReceiverRejectsWrongBrowser(t *testing.T) {
	idp := newStubIdP(t)
	c := newTestConnector(idp)
	params, _ := startLogin(t, c)
	stubNonce = params.Get("nonce")

	// No cookie at all (e.g. victim's browser hit with attacker's URL).
	_, err := callback(c, params.Get("state"), nil)
	assert.ErrorContains(t, err, "not bound to this browser")

	// Wrong cookie value.
	params2, _ := startLogin(t, c)
	stubNonce = params2.Get("nonce")
	_, err = callback(c, params2.Get("state"), &http.Cookie{Name: bindCookieName, Value: "forged"})
	assert.ErrorContains(t, err, "not bound to this browser")
}

func TestReceiverSendsPKCEVerifier(t *testing.T) {
	idp := newStubIdP(t)
	c := newTestConnector(idp)
	params, cookie := startLogin(t, c)
	stubNonce = params.Get("nonce")

	_, err := callback(c, params.Get("state"), cookie)
	require.NoError(t, err)

	verifier := idp.lastTokenRequest.Get("code_verifier")
	require.NotEmpty(t, verifier, "token exchange must carry code_verifier")
	sum := sha256.Sum256([]byte(verifier))
	assert.Equal(t, params.Get("code_challenge"), base64.RawURLEncoding.EncodeToString(sum[:]),
		"verifier must match the challenge sent to /authorize")
}

func TestReceiverValidatesIDToken(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(idp *stubIdP)
		errPart string
	}{
		{"tampered signature", func(i *stubIdP) { i.tamperSignature = true }, "signature invalid"},
		{"wrong audience", func(i *stubIdP) {
			i.idTokenOverride = func(c map[string]any) { c["aud"] = "someone-else" }
		}, "audience"},
		{"expired", func(i *stubIdP) {
			i.idTokenOverride = func(c map[string]any) { c["exp"] = time.Now().Add(-time.Hour).Unix() }
		}, "expired"},
		{"wrong issuer", func(i *stubIdP) {
			i.idTokenOverride = func(c map[string]any) { c["iss"] = "https://evil.example" }
		}, "issuer"},
		{"nonce mismatch", func(i *stubIdP) {
			i.idTokenOverride = func(c map[string]any) { c["nonce"] = "stale-nonce" }
		}, "nonce mismatch"},
		{"missing id_token", func(i *stubIdP) { i.omitIDToken = true }, "missing id_token"},
		{"empty subject", func(i *stubIdP) {
			i.idTokenOverride = func(c map[string]any) { c["sub"] = "" }
		}, "no subject"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idp := newStubIdP(t)
			c := newTestConnector(idp)
			params, cookie := startLogin(t, c)
			stubNonce = params.Get("nonce")
			tc.mutate(idp)

			_, err := callback(c, params.Get("state"), cookie)
			assert.ErrorContains(t, err, tc.errPart)
		})
	}
}

func TestReceiverRejectsUserinfoFailures(t *testing.T) {
	idp := newStubIdP(t)
	c := newTestConnector(idp)
	params, cookie := startLogin(t, c)
	stubNonce = params.Get("nonce")
	idp.userinfoStatus = http.StatusInternalServerError

	_, err := callback(c, params.Get("state"), cookie)
	assert.ErrorContains(t, err, "userinfo failed (500)")

	// Subject swap between ID token and userinfo must be rejected.
	idp2 := newStubIdP(t)
	c2 := newTestConnector(idp2)
	params2, cookie2 := startLogin(t, c2)
	stubNonce = params2.Get("nonce")
	idp2.userinfoSubject = "user-2"
	_, err = callback(c2, params2.Get("state"), cookie2)
	assert.ErrorContains(t, err, "does not match id_token subject")
}

func TestReceiverHappyPathAndEmailVerification(t *testing.T) {
	idp := newStubIdP(t)
	c := newTestConnector(idp)
	params, cookie := startLogin(t, c)
	stubNonce = params.Get("nonce")

	req := httptest.NewRequest("GET", "http://answer.example/receiver?code=abc&state="+url.QueryEscape(params.Get("state")), nil)
	req.AddCookie(cookie)
	ctx, _ := ginCtx(req)
	info, err := c.ConnectorReceiver(ctx, "http://answer.example/receiver")
	require.NoError(t, err)
	assert.Equal(t, "user-1", info.ExternalID)
	assert.Equal(t, "alice", info.Username)
	assert.True(t, info.UsernameAuthoritative)
	assert.Equal(t, "alice@example.com", info.Email, "verified email is forwarded")

	// email_verified=false → email must be withheld so the core can never
	// bind an unverified address to an existing account.
	idp2 := newStubIdP(t)
	idp2.emailVerified = false
	c2 := newTestConnector(idp2)
	params2, cookie2 := startLogin(t, c2)
	stubNonce = params2.Get("nonce")
	req2 := httptest.NewRequest("GET", "http://answer.example/receiver?code=abc&state="+url.QueryEscape(params2.Get("state")), nil)
	req2.AddCookie(cookie2)
	ctx2, _ := ginCtx(req2)
	info2, err := c2.ConnectorReceiver(ctx2, "http://answer.example/receiver")
	require.NoError(t, err)
	assert.Empty(t, info2.Email, "unverified email must not be forwarded")

	// The ID token and userinfo must name the same address; a mismatch is
	// treated like an unverified email.
	idp3 := newStubIdP(t)
	idp3.idTokenOverride = func(c map[string]any) { c["email"] = "someone-else@example.com" }
	c3 := newTestConnector(idp3)
	params3, cookie3 := startLogin(t, c3)
	stubNonce = params3.Get("nonce")
	req3 := httptest.NewRequest("GET", "http://answer.example/receiver?code=abc&state="+url.QueryEscape(params3.Get("state")), nil)
	req3.AddCookie(cookie3)
	ctx3, _ := ginCtx(req3)
	info3, err := c3.ConnectorReceiver(ctx3, "http://answer.example/receiver")
	require.NoError(t, err)
	assert.Empty(t, info3.Email, "email mismatch between id_token and userinfo must not be forwarded")
	assert.Equal(t, "user-1", info2.ExternalID)
}

func TestPendingStoreExpiry(t *testing.T) {
	s := newPendingStore()
	s.put("s1", pendingAuth{expires: time.Now().Add(-time.Second)})
	_, ok := s.take("s1")
	assert.False(t, ok, "expired entries must not validate")

	s.put("s2", pendingAuth{expires: time.Now().Add(time.Minute)})
	_, ok = s.take("s2")
	assert.True(t, ok)
	_, ok = s.take("s2")
	assert.False(t, ok, "single-use")
}
