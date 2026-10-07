// SPDX-License-Identifier: Apache-2.0
package secrets

import (
	"context"
	"errors"
	"github.com/danieljoos/wincred"
	"github.com/shruggietech/insonic/internal/contracts"
)

type windowsNative struct{ service string }

func newNative(service string) nativeBackend { return windowsNative{service} }
func (n windowsNative) available(ctx context.Context) error {
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	// A random absent target tests API access without retrieving a saved value.
	_, e := wincred.GetGenericCredential(n.service + "/probe/" + contracts.ID())
	if errors.Is(e, wincred.ErrElementNotFound) {
		return nil
	}
	return contracts.Fail("unavailable")
}
func (n windowsNative) get(ctx context.Context, id string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	c, e := wincred.GetGenericCredential(n.service + "/" + id)
	if errors.Is(e, wincred.ErrElementNotFound) {
		return nil, contracts.Fail("not_found")
	}
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	value := append([]byte(nil), c.CredentialBlob...)
	wipe(c.CredentialBlob)
	return value, nil
}
func (n windowsNative) put(ctx context.Context, id string, value []byte) error {
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	c := wincred.NewGenericCredential(n.service + "/" + id)
	c.UserName = id
	c.CredentialBlob = append([]byte(nil), value...)
	defer wipe(c.CredentialBlob)
	if c.Write() != nil {
		return contracts.Fail("unavailable")
	}
	return nil
}
func (n windowsNative) remove(ctx context.Context, id string) error {
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	c := wincred.NewGenericCredential(n.service + "/" + id)
	e := c.Delete()
	if e != nil && !errors.Is(e, wincred.ErrElementNotFound) {
		return contracts.Fail("unavailable")
	}
	return nil
}
