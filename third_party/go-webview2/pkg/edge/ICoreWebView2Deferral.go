//go:build windows

// insonic maintained change: release the owned deferral reference instead of adding one.
// See NOTICE.insonic and the repository source manifest for patch authority.

package edge

import (
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type ICoreWebView2DeferralVtbl struct {
	_IUnknownVtbl
	Complete ComProc
}

type ICoreWebView2Deferral struct {
	Vtbl *ICoreWebView2DeferralVtbl
}

// AddRef increments the reference count of ICoreWebView2Deferral interface
func (i *ICoreWebView2Deferral) AddRef() error {
	_, _, err := i.Vtbl.AddRef.Call(uintptr(unsafe.Pointer(i)))
	if err != nil && !errors.Is(err, windows.ERROR_SUCCESS) {
		return err
	}

	return nil
}

// Release decrements the reference count of ICoreWebView2Deferral interface
func (i *ICoreWebView2Deferral) Release() error {
	retireBootstrapResponse(uintptr(unsafe.Pointer(i)), true)
	_, _, err := i.Vtbl.Release.Call(uintptr(unsafe.Pointer(i)))
	if err != nil && !errors.Is(err, windows.ERROR_SUCCESS) {
		return err
	}

	return nil
}

func (i *ICoreWebView2Deferral) Complete() error {
	recordBootstrapResponseCall(uintptr(unsafe.Pointer(i)), bootstrapResponseCompletion, false, 0, 0)
	hr, _, _ := i.Vtbl.Complete.Call(
		uintptr(unsafe.Pointer(i)),
	)
	recordBootstrapResponseCall(uintptr(unsafe.Pointer(i)), bootstrapResponseCompletion, true, hr, 0)
	if windows.Handle(hr) != windows.S_OK {
		return syscall.Errno(hr)
	}
	return nil
}
