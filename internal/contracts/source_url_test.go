// SPDX-License-Identifier: Apache-2.0
package contracts

import "testing"

func TestSourceQuerySelectorsAndCredentialReferences(t *testing.T) {
	good := "https://example.org/media?id=fixture&page=2&download=true"
	u, e := SourceURL(good)
	if e != nil || u.String() != good {
		t.Fatal("ordinary selector lost")
	}
	for _, bad := range []string{"https://user:password@example.org/media", "https://example.org/media?token=secret", "https://example.org/media?X-Amz-Signature=secret", "https://example.org/media?api_key=secret", "https://example.org/media#private"} {
		if _, e = SourceURL(bad); e == nil {
			t.Fatal("authentication locator accepted")
		}
	}
}
