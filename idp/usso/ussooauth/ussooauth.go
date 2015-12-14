// Copyright 2015 Canonical Ltd.

// Pacakge ussooauth is an identity provider that authenticates against
// Ubuntu SSO using OAuth.
package ussooauth

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"regexp"

	"gopkg.in/errgo.v1"

	"github.com/CanonicalLtd/blues-identity/config"
	"github.com/CanonicalLtd/blues-identity/idp"
	"github.com/CanonicalLtd/blues-identity/idp/idputil"
)

func init() {
	config.RegisterIDP("usso_oauth", func(func(interface{}) error) (idp.IdentityProvider, error) {
		return IdentityProvider, nil
	})
}

// IdentityProvider is an idp.IdentityProvider that provides
// authentication via Ubuntu SSO using OAuth.
var IdentityProvider idp.IdentityProvider = (*identityProvider)(nil)

const (
	ussoURL = "https://login.ubuntu.com"
)

// identityProvider allows login using request signing with
// Ubuntu SSO OAuth tokens.
type identityProvider struct{}

// Name gives the name of the identity provider (usso_oauth).
func (*identityProvider) Name() string {
	return "usso_oauth"
}

// Description gives a description of the identity provider.
func (*identityProvider) Description() string {
	return "Ubuntu SSO OAuth"
}

// Interactive specifies that this identity provider is not interactive.
func (*identityProvider) Interactive() bool {
	return false
}

// URL gets the login URL to use this identity provider.
func (*identityProvider) URL(c idp.URLContext, waitID string) (string, error) {
	callback := c.URL("/oauth")
	if waitID != "" {
		callback += "?waitid=" + waitID
	}
	return callback, nil
}

// Handle handles the Ubuntu SSO OAuth login process.
func (*identityProvider) Handle(c idp.Context) {
	id, err := verifyOAuthSignature(c.RequestURL(), c.Params().Request)
	if err != nil {
		c.LoginFailure(err)
		return
	}
	u, err := c.FindUserByExternalId(id)
	if err != nil {
		c.LoginFailure(errgo.Notef(err, "cannot get user details for %q", id))
		return
	}
	idputil.LoginUser(c, u)
}

var consumerKeyRegexp = regexp.MustCompile(`oauth_consumer_key="([^"]*)"`)

// verifyOAuthSignature verifies with Ubuntu SSO that the request is correctly
// signed.
func verifyOAuthSignature(requestURL string, req *http.Request) (string, error) {
	req.ParseForm()
	u, err := url.Parse(requestURL)
	if err != nil {
		return "", errgo.Notef(err, "cannot parse request URL")
	}
	u.RawQuery = ""
	request := struct {
		URL           string `json:"http_url"`
		Method        string `json:"http_method"`
		Authorization string `json:"authorization"`
		QueryString   string `json:"query_string"`
	}{
		URL:           u.String(),
		Method:        req.Method,
		Authorization: req.Header.Get("Authorization"),
		QueryString:   req.Form.Encode(),
	}
	buf, err := json.Marshal(request)
	if err != nil {
		return "", errgo.Notef(err, "cannot marshal request")
	}
	resp, err := http.Post(ussoURL+"/api/v2/requests/validate", "application/json", bytes.NewReader(buf))
	if err != nil {
		return "", errgo.Mask(err)
	}
	defer resp.Body.Close()
	t, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		return "", errgo.Newf("bad content type %q", resp.Header.Get("Content-Type"))
	}
	if t != "application/json" {
		return "", errgo.Newf("unexpected response type %q", t)
	}
	var validated struct {
		IsValid bool   `json:"is_valid"`
		Error   string `json:"error"`
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(data, &validated); err != nil {
		return "", errgo.Mask(err)
	}
	if validated.Error != "" {
		return "", errgo.Newf("cannot validate OAuth credentials: %s", validated.Error)
	}
	if !validated.IsValid {
		return "", errgo.Newf("invalid OAuth credentials")
	}
	consumerKey := consumerKeyRegexp.FindStringSubmatch(req.Header.Get("Authorization"))
	if len(consumerKey) != 2 {
		return "", errgo.Newf("no customer key in authorization")
	}
	return ussoURL + "/+id/" + consumerKey[1], nil
}