// SPDX-License-Identifier: Apache-2.0
package models

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"net/http"
	"regexp"
	"strings"
)

var immutableHF = regexp.MustCompile(`^/[^/]+/[^/]+/resolve/[a-f0-9]{40}/.+$`)

func hfCDN(host string) bool {
	return host == "huggingface.co" || host == "cdn-lfs.huggingface.co" || host == "cdn-lfs-us-1.hf.co" || host == "cdn-lfs-eu-1.hf.co" || host == "cdn-lfs.hf.co" || host == "cas-bridge.xethub.hf.co" || host == "us.aws.cdn.hf.co" || host == "us.gcp.cdn.hf.co"
}

// Signed CDN queries are transient request state, never model authority or output.
func boundedRedirect(req *http.Request, via []*http.Request) error {
	if len(via) == 0 || len(via) > 8 {
		return contracts.Fail("unavailable")
	}
	origin := via[0].URL
	if req.URL.Host != origin.Host {
		req.Header.Del("Authorization")
		req.Header.Del("Cookie")
	}
	if req.URL.User != nil || req.URL.Fragment != "" || req.URL.Scheme != origin.Scheme {
		return contracts.Fail("unavailable")
	}
	if req.URL.Host == origin.Host {
		return nil
	}
	if req.URL.Scheme != "https" || req.URL.Port() != "" && req.URL.Port() != "443" {
		return contracts.Fail("unavailable")
	}
	host := strings.ToLower(req.URL.Hostname())
	if origin.Hostname() == "huggingface.co" && immutableHF.MatchString(origin.Path) && hfCDN(host) {
		return nil
	}
	if origin.Hostname() == "github.com" && (strings.HasPrefix(origin.Path, "/astral-sh/python-build-standalone/releases/download/") || strings.HasPrefix(origin.Path, "/astral-sh/uv/releases/download/")) && (host == "release-assets.githubusercontent.com" || host == "objects.githubusercontent.com") {
		return nil
	}
	if (origin.Hostname() == "download.pytorch.org" || origin.Hostname() == "download-r2.pytorch.org") && strings.HasPrefix(origin.Path, "/whl/cpu/") && (host == "download.pytorch.org" || host == "download-r2.pytorch.org") {
		return nil
	}
	return contracts.Fail("unavailable")
}
