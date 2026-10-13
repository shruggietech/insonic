// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"net"
	"testing"
)

func TestPostgreSQLTransportMeaning(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1", "localhost"} {
		c, e := PostgreSQLBaseConfig(PostgreSQLConfig{Host: host, Port: 5432, Database: "fixture", Schema: "fixture", TLSMode: "local"})
		if e != nil || c.TLSConfig != nil || net.ParseIP(c.Host) == nil || !net.ParseIP(c.Host).IsLoopback() {
			t.Fatal(host, e)
		}
	}
	if _, e := PostgreSQLBaseConfig(PostgreSQLConfig{Host: "remote.example", Port: 5432, Database: "fixture", Schema: "fixture", TLSMode: "local"}); e == nil {
		t.Fatal("remote mislabeled local")
	}
	for _, mode := range []string{"verify-full", "insecure"} {
		c, e := PostgreSQLBaseConfig(PostgreSQLConfig{Host: "remote.example", Port: 5432, Database: "fixture", Schema: "fixture", TLSMode: mode})
		if e != nil || (c.TLSConfig != nil) != (mode == "verify-full") || c.Fallbacks != nil {
			t.Fatal(mode, e)
		}
	}
}
