package runtime

import (
	"github.com/shruggietech/insonic/internal/workspace"
	"golang.org/x/sys/windows"
	"strings"
	"testing"
)

func TestPipeDACLOnlyAdmitsCurrentUser(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "security")
	if err != nil {
		t.Fatal(err)
	}
	_, security, err := endpoint(w)
	if err != nil {
		t.Fatal(err)
	}
	user, _ := windows.GetCurrentProcessToken().GetTokenUser()
	if security != "D:P(A;;GA;;;"+user.User.Sid.String()+")" || strings.Contains(security, ";;;WD") {
		t.Fatal("other-account pipe access permitted")
	}
}
