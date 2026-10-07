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
		case "secret", "token", "auth", "authorization", "pwd", "signature", "sig", "key", "credential", "credentials", "xamzcredential", "xamzsignature", "xamzsecuritytoken", "xgoogcredential", "xgoogsignature":
			return nil, Fail("invalid_request")
		}
		for _, suffix := range []string{"password", "passwd", "passphrase", "apikey", "apitoken", "accesstoken", "refreshtoken", "authtoken", "bearertoken", "clientsecret", "privatekey", "secretkey", "accesskey", "securitytoken"} {
			if strings.HasSuffix(key, suffix) {
				return nil, Fail("invalid_request")
			}
		}
	}
	return u, nil
}
