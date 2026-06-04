package fastgate_connector

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/apache/answer/plugin"
)

type Connector struct {
	Config *ConnectorConfig
}

type ConnectorConfig struct {
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func init() {
	plugin.Register(&Connector{
		Config: &ConnectorConfig{},
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

	params := url.Values{
		"client_id":     {c.Config.ClientID},
		"redirect_uri":  {receiverURL},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {receiverURL},
	}
	return authURL + "?" + params.Encode()
}

func (c *Connector) ConnectorReceiver(ctx *plugin.GinContext, receiverURL string) (userInfo plugin.ExternalLoginUserInfo, err error) {
	code := ctx.Query("code")
	if code == "" {
		return userInfo, fmt.Errorf("missing authorization code")
	}

	// Exchange code for tokens
	tokenURL := c.Config.Issuer + "/token"
	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {receiverURL},
		"client_id":    {c.Config.ClientID},
	}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return userInfo, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.Config.ClientID, c.Config.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return userInfo, fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

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

	// Fetch user info
	uiReq, err := http.NewRequest("GET", c.Config.Issuer+"/userinfo", nil)
	if err != nil {
		return userInfo, fmt.Errorf("create userinfo request: %w", err)
	}
	uiReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	uiResp, err := http.DefaultClient.Do(uiReq)
	if err != nil {
		return userInfo, fmt.Errorf("userinfo request: %w", err)
	}
	defer uiResp.Body.Close()

	var claims struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Picture           string `json:"picture"`
	}
	if err := json.NewDecoder(uiResp.Body).Decode(&claims); err != nil {
		return userInfo, fmt.Errorf("decode userinfo: %w", err)
	}

	username := claims.PreferredUsername
	if username == "" {
		username = claims.Email
	}

	return plugin.ExternalLoginUserInfo{
		ExternalID:  claims.Sub,
		DisplayName: claims.Name,
		Username:    username,
		Email:       claims.Email,
		Avatar:      claims.Picture,
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
