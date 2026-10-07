// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
)

func TestAcquisitionCLITransportAndResourceOptions(t *testing.T) {
	credential := contracts.ID()
	_, _, raw, e := parseDomain([]string{"media", "import", "http://127.0.0.1/source.wav", "--credential-id", credential, "--local-http", "--acquisition-max-bytes", "1024", "--acquisition-timeout-ms", "5000"})
	if e != nil {
		t.Fatal(e)
	}
	var req library.ImportRequest
	if json.Unmarshal(raw, &req) != nil || req.Defaults.LocalHTTP == nil || !*req.Defaults.LocalHTTP || req.Defaults.AcquisitionMaxBytes == nil || *req.Defaults.AcquisitionMaxBytes != 1024 || req.Defaults.AcquisitionTimeoutMS == nil || *req.Defaults.AcquisitionTimeoutMS != 5000 {
		t.Fatal(string(raw))
	}
	for _, args := range [][]string{
		{"media", "import", "http://127.0.0.1/source.wav", "--credential-id", credential},
		{"media", "import", "http://example.test/source.wav", "--credential-id", credential, "--local-http"},
		{"media", "import", "https://example.test/source.wav", "--acquisition-max-bytes", "0"},
		{"media", "import", "https://example.test/source.wav", "--acquisition-timeout-ms", "-1"},
		{"media", "import", "https://example.test/source.wav", "--acquisition-timeout-ms", "9223372036854775807"},
	} {
		if _, _, _, e := parseDomain(args); e == nil {
			t.Fatal("invalid acquisition admitted", args)
		}
	}
	if _, _, _, e := parseDomain([]string{"media", "import", "http://example.test/source.wav"}); e != nil {
		t.Fatal("ordinary noncredential HTTP rejected", e)
	}
}
