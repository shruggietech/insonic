// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"net/url"
	"strings"
	"unicode"
)

// SourceURL preserves ordinary source selectors while keeping authentication
// in credential references. Callers enforce their adapter's transport policy.
func SourceURL(raw string) (*url.URL, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || strings.ContainsAny(raw, "\r\n\x00") {
		return nil, Fail("invalid_request")
	}
	q, e := url.ParseQuery(u.RawQuery)
	if e != nil {
		return nil, Fail("invalid_request")
	}
	for key := range q {
		key = strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, key)
		switch key {
		case "secret", "token", "auth", "authorization", "authentication", "pwd", "signature", "sig", "key", "credential", "credentials", "jwt", "session", "sid", "jsessionid", "phpsessid", "connectsid", "aspxauth", "oauth", "oauth2", "sso", "assertion", "samlresponse", "xamzcredential", "xamzsignature", "xamzsecuritytoken", "xgoogcredential", "xgoogsignature":
			return nil, Fail("invalid_request")
		}
		for _, suffix := range []string{"password", "passwd", "passphrase", "apikey", "apitoken", "accesstoken", "refreshtoken", "authtoken", "authenticationtoken", "authorizationtoken", "bearertoken", "clientsecret", "privatekey", "secretkey", "accesskey", "securitytoken", "jwt", "jwttoken", "sessiontoken", "sessionid", "sessionkey", "sessionsecret", "sessionticket", "authsession", "authsid", "oauthtoken", "oauth2token", "oauthcode", "oauthsecret", "idtoken", "identitytoken", "ssotoken", "authcode", "authorizationcode", "samlassertion"} {
			if strings.HasSuffix(key, suffix) {
				return nil, Fail("invalid_request")
			}
		}
	}
	return u, nil
}
