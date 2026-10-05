//go:build !windows

package runtime

import (
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"testing"
)

func TestSocketCurrentUserPermissions(t *testing.T) {
	w, _ := workspace.Init(t.TempDir(), "security")
	listener, cleanup, err := listen(w)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	defer listener.Close()
	file, _ := endpoint(w)
	stat, err := os.Stat(file)
	if err != nil || stat.Mode().Perm() != 0600 {
		t.Fatal("socket permits another account")
	}
}
