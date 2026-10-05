// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/Microsoft/go-winio"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"golang.org/x/sys/windows"
	"net"
	"strings"
)

func endpoint(w *workspace.Workspace) (string, string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", "", contracts.Fail("unavailable")
	}
	sid := user.User.Sid.String()
	hash := sha256.Sum256([]byte(strings.ToLower(w.Root)))
	return `\\.\pipe\insonic-` + sid + fmt.Sprintf("-%x", hash[:16]), "D:P(A;;GA;;;" + sid + ")", nil
}
func listen(w *workspace.Workspace) (net.Listener, func(), error) {
	name, security, err := endpoint(w)
	if err != nil {
		return nil, nil, err
	}
	listener, err := winio.ListenPipe(name, &winio.PipeConfig{SecurityDescriptor: security, InputBufferSize: 4096, OutputBufferSize: 4096})
	if err != nil {
		return nil, nil, contracts.Fail("unavailable")
	}
	return listener, func() { listener.Close() }, nil
}
func dial(ctx context.Context, w *workspace.Workspace) (net.Conn, error) {
	name, _, err := endpoint(w)
	if err != nil {
		return nil, err
	}
	return winio.DialPipeContext(ctx, name)
}
