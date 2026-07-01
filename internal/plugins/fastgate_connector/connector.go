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
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/apache/answer/plugin"
	"github.com/segmentfault/pacman/log"
)

type Connector struct {
	Config  *ConnectorConfig
	pending *pendingStore
	jwks    *jwksCache
	client  *http.Client
}

type ConnectorConfig struct {
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func init() {
	plugin.Register(&Connector{
		Config:  &ConnectorConfig{},
		pending: newPendingStore(),
		jwks:    &jwksCache{},
		client:  &http.Client{Timeout: 10 * time.Second},
	})
}

func (c *Connector) Info() plugin.Info {
	return plugin.Info{
		Name:        plugin.MakeTranslator("Fastgate"),
		SlugName:    "fastgate-connector",
		Description: plugin.MakeTranslator("Login with Fastgate (OIDC)"),
		Version:     "0.1.0",
	}
}

func (c *Connector) ConnectorLogoSVG() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><polyline points="10 17 15 12 10 7"/><line x1="15" y1="12" x2="3" y2="12"/></svg>`
}

func (c *Connector) ConnectorName() plugin.Translator {
	return plugin.MakeTranslator("Fastgate")
}

func (c *Connector) ConnectorSlugName() string {
	return "fastgate-connector"
}

func (c *Connector) ConnectorSender(ctx *plugin.GinContext, receiverURL string) (redirectURL string) {
	issuer := c.Config.Issuer
	authURL := issuer + "/authorize"

	state, err1 := randomToken()
	nonce, err2 := randomToken()
	verifier, err3 := randomToken()
	bind, err4 := randomToken()
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		log.Errorf("fastgate connector: generating login secrets failed: %v %v %v %v", err1, err2, err3, err4)
		return ""
	}

	c.pending.put(state, pendingAuth{
		nonce:    nonce,
		verifier: verifier,
		bind:     bind,
		expires:  time.Now().Add(pendingTTL),
	})

	// Bind the attempt to this browser: the callback must present the same
	// cookie, so a stolen/forced callback URL can't complete elsewhere.
	secure := ctx.Request.TLS != nil || strings.EqualFold(ctx.GetHeader("X-Forwarded-Proto"), "https")
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(bindCookieName, bind, int(pendingTTL.Seconds()), "/", "", secure, true)

	params := url.Values{
		"client_id":             {c.Config.ClientID},
		"redirect_uri":          {receiverURL},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {pkceChallengeS256(verifier)},
		"code_challenge_method": {"S256"},
	}
	return authURL + "?" + params.Encode()
}

func (c *Connector) ConnectorReceiver(ctx *plugin.GinContext, receiverURL string) (userInfo plugin.ExternalLoginUserInfo, err error) {
	code := ctx.Query("code")
	if code == "" {
		return userInfo, fmt.Errorf("missing authorization code")
	}

	// The state must be one we issued (single-use, unexpired) and the
	// browser must carry the bind cookie from the same attempt.
	state := ctx.Query("state")
	if state == "" {
		return userInfo, fmt.Errorf("missing state")
	}
	pend, ok := c.pending.take(state)
	if !ok {
		return userInfo, fmt.Errorf("unknown, expired or replayed state")
	}
	bind, err := ctx.Cookie(bindCookieName)
	if err != nil || subtle.ConstantTimeCompare([]byte(bind), []byte(pend.bind)) != 1 {
		return userInfo, fmt.Errorf("login attempt not bound to this browser")
	}
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(bindCookieName, "", -1, "/", "", false, true)

	// Exchange code for tokens
	tokenURL := c.Config.Issuer + "/token"
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {receiverURL},
		"client_id":     {c.Config.ClientID},
		"code_verifier": {pend.verifier},
	}

	req, err := http.NewRequestWithContext(ctx.Request.Context(), "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return userInfo, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.Config.ClientID, c.Config.ClientSecret)

	resp, err := c.client.Do(req)
	if err != nil {
		return userInfo, fmt.Errorf("token exchange: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return userInfo, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, body)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return userInfo, fmt.Errorf("decode token response: %w", err)
	}
	if tokenResp.IDToken == "" {
		return userInfo, fmt.Errorf("token response missing id_token")
	}

	// The ID token is the authenticated identity assertion: verify its
	// signature against the issuer's JWKS plus iss/aud/exp/iat/nonce.
	idClaims, err := verifyIDToken(c.client, c.jwks, c.Config.Issuer, c.Config.ClientID, tokenResp.IDToken, pend.nonce)
	if err != nil {
		return userInfo, fmt.Errorf("id_token rejected: %w", err)
	}

	// Fetch user info (profile fields the ID token doesn't carry).
	uiReq, err := http.NewRequestWithContext(ctx.Request.Context(), "GET", c.Config.Issuer+"/userinfo", nil)
	if err != nil {
		return userInfo, fmt.Errorf("create userinfo request: %w", err)
	}
	uiReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	uiResp, err := c.client.Do(uiReq)
	if err != nil {
		return userInfo, fmt.Errorf("userinfo request: %w", err)
	}
	defer func() { _ = uiResp.Body.Close() }()

	if uiResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(uiResp.Body)
		return userInfo, fmt.Errorf("userinfo failed (%d): %s", uiResp.StatusCode, body)
	}

	var claims struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Picture           string `json:"picture"`
	}
	if err := json.NewDecoder(uiResp.Body).Decode(&claims); err != nil {
		return userInfo, fmt.Errorf("decode userinfo: %w", err)
	}
	if claims.Sub != idClaims.Sub {
		return userInfo, fmt.Errorf("userinfo subject %q does not match id_token subject", claims.Sub)
	}

	// preferred_username is the fastgate handle: upstream-validated,
	// globally unique, owned by the IdP. Pass it through verbatim and
	// flag it so the service layer never transforms or dedup-suffixes.
	// Fall back to email only if the IdP didn't send a handle (shouldn't
	// happen with fastgate, but keeps the connector resilient if the
	// handle requirement is ever relaxed upstream).
	username := claims.PreferredUsername
	authoritative := username != ""
	if username == "" {
		username = claims.Email
	}

	// The core binds existing accounts by email alone, so per the plugin
	// contract the email is only forwarded when the IdP asserts it is
	// verified (fastgate always does; both sources must agree).
	email := claims.Email
	if !claims.EmailVerified || !idClaims.EmailVerified {
		email = ""
	}

	return plugin.ExternalLoginUserInfo{
		ExternalID:            idClaims.Sub,
		DisplayName:           claims.Name,
		Username:              username,
		Email:                 email,
		Avatar:                claims.Picture,
		UsernameAuthoritative: authoritative,
	}, nil
}

func (c *Connector) ConfigFields() []plugin.ConfigField {
	return []plugin.ConfigField{
		{
			Name:        "issuer",
			Type:        plugin.ConfigTypeInput,
			Title:       plugin.MakeTranslator("Issuer URL"),
			Description: plugin.MakeTranslator("Fastgate base URL (e.g. http://localhost:8089)"),
			Required:    true,
			UIOptions: plugin.ConfigFieldUIOptions{
				InputType: plugin.InputTypeText,
			},
			Value: c.Config.Issuer,
		},
		{
			Name:        "client_id",
			Type:        plugin.ConfigTypeInput,
			Title:       plugin.MakeTranslator("Client ID"),
			Description: plugin.MakeTranslator("OIDC client ID from fgctl"),
			Required:    true,
			UIOptions: plugin.ConfigFieldUIOptions{
				InputType: plugin.InputTypeText,
			},
			Value: c.Config.ClientID,
		},
		{
			Name:        "client_secret",
			Type:        plugin.ConfigTypeInput,
			Title:       plugin.MakeTranslator("Client Secret"),
			Description: plugin.MakeTranslator("OIDC client secret from fgctl"),
			Required:    true,
			UIOptions: plugin.ConfigFieldUIOptions{
				InputType: plugin.InputTypePassword,
			},
			Value: c.Config.ClientSecret,
		},
	}
}

func (c *Connector) ConfigReceiver(config []byte) error {
	conf := &ConnectorConfig{}
	if err := json.Unmarshal(config, conf); err != nil {
		return err
	}
	c.Config = conf
	return nil
}

// AfterLogin reports the Answer user_id back to fastgate's directory so the
// IDP can resolve cross-app references (e.g. "find this user in Zulip") and
// fan deactivations out to the right local account. Best-effort: errors are
// logged but never block the user signing in.
func (c *Connector) AfterLogin(ctx context.Context, externalID, localUserID string) error {
	if c.Config == nil || c.Config.Issuer == "" || c.Config.ClientID == "" {
		return nil
	}
	endpoint := fmt.Sprintf("%s/directory/users/%s/apps/%s/identity",
		strings.TrimRight(c.Config.Issuer, "/"),
		url.PathEscape(externalID),
		url.PathEscape(c.Config.ClientID))

	body, err := json.Marshal(map[string]string{"app_user_id": localUserID})
	if err != nil {
		return fmt.Errorf("encode identity body: %w", err)
	}

	rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(rctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build identity request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.Config.ClientID, c.Config.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post identity: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	respBody, _ := io.ReadAll(resp.Body)
	log.Warnf("fastgate directory identity report rejected (%d): %s", resp.StatusCode, respBody)
	return fmt.Errorf("identity report status %d", resp.StatusCode)
}
