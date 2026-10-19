//go:build windows

// SPDX-License-Identifier: Apache-2.0
package edge

import (
	"unsafe"

	"github.com/wailsapp/go-webview2/pkg/combridge"
)

// Each callback has one Go-owned lease and any references retained by native
// WebView2. The bridge keeps the target rooted until the final native Release;
// its native allocation exposes only a vtable pointer, never a Go interface.
type nativeCallbackLease interface{ Close() error }

type environmentCompletedCallback interface {
	EnvironmentCompleted(uintptr, *ICoreWebView2Environment) uintptr
}
type controllerCompletedCallback interface {
	CreateCoreWebView2ControllerCompleted(uintptr, *ICoreWebView2Controller) uintptr
}
type webMessageCallback interface {
	MessageReceived(*ICoreWebView2, *ICoreWebView2WebMessageReceivedEventArgs) uintptr
}
type permissionCallback interface {
	PermissionRequested(*ICoreWebView2, *iCoreWebView2PermissionRequestedEventArgs) uintptr
}
type resourceCallback interface {
	WebResourceRequested(*ICoreWebView2, *ICoreWebView2WebResourceRequestedEventArgs) uintptr
}
type acceleratorCallback interface {
	AcceleratorKeyPressed(*ICoreWebView2Controller, *ICoreWebView2AcceleratorKeyPressedEventArgs) uintptr
}
type navigationCallback interface {
	NavigationCompleted(*ICoreWebView2, *ICoreWebView2NavigationCompletedEventArgs) uintptr
}
type processFailureCallback interface {
	ProcessFailed(*ICoreWebView2, *ICoreWebView2ProcessFailedEventArgs) uintptr
}
type fullscreenCallback interface {
	ContainsFullScreenElementChanged(*ICoreWebView2, *ICoreWebView2ContainsFullScreenElementChangedEventArgs) uintptr
}

// IIDs come from Microsoft.Web.WebView2 1.0.3537.50, build/native/include/WebView2.h.
// They identify the callback interfaces, not their event-argument interfaces.
func init() {
	combridge.RegisterVTable[combridge.IUnknown, environmentCompletedCallback]("{4e8a3389-c9d8-4bd2-b6b5-124fee6cc14d}", invokeEnvironmentCompleted)
	combridge.RegisterVTable[combridge.IUnknown, controllerCompletedCallback]("{6c4819f3-c9b7-4260-8127-c9f5bde7f68c}", invokeControllerCompleted)
	combridge.RegisterVTable[combridge.IUnknown, webMessageCallback]("{57213f19-00e6-49fa-8e07-898ea01ecbd2}", invokeWebMessage)
	combridge.RegisterVTable[combridge.IUnknown, permissionCallback]("{15e1c6a3-c72a-4df3-91d7-d097fbec6bfd}", invokePermission)
	combridge.RegisterVTable[combridge.IUnknown, resourceCallback]("{ab00b74c-15f1-4646-80e8-e76341d25d71}", invokeResource)
	combridge.RegisterVTable[combridge.IUnknown, acceleratorCallback]("{b29c7e28-fa79-41a8-8e44-65811c76dcb2}", invokeAccelerator)
	combridge.RegisterVTable[combridge.IUnknown, navigationCallback]("{d33a35bf-1c49-4f98-93ab-006e0533fe1c}", invokeNavigation)
	combridge.RegisterVTable[combridge.IUnknown, processFailureCallback]("{79e0aea4-990b-42d9-aa1d-0fcc2e5bc7f1}", invokeProcessFailure)
	combridge.RegisterVTable[combridge.IUnknown, fullscreenCallback]("{e45d98b1-afef-45be-8baf-6c7728867f73}", invokeFullscreen)
}

func retainNativeCallback[T combridge.IUnknown](e *Chromium, target T) uintptr {
	owner := combridge.New[T](target)
	e.callbackLeases = append(e.callbackLeases, owner)
	return owner.Ref()
}

// Prepare only after the native window acquires its lifetime owner. Merely
// constructing a Chromium must not leave a registry-rooted reference cycle.
func (e *Chromium) prepareOwnedCallbacks() {
	if e.shuttingDown || len(e.callbackLeases) != 0 {
		return
	}
	e.envCompleted = (*iCoreWebView2CreateCoreWebView2EnvironmentCompletedHandler)(unsafe.Pointer(retainNativeCallback[environmentCompletedCallback](e, e)))
	e.controllerCompleted = (*iCoreWebView2CreateCoreWebView2ControllerCompletedHandler)(unsafe.Pointer(retainNativeCallback[controllerCompletedCallback](e, e)))
	e.webMessageReceived = (*iCoreWebView2WebMessageReceivedEventHandler)(unsafe.Pointer(retainNativeCallback[webMessageCallback](e, e)))
	e.permissionRequested = (*iCoreWebView2PermissionRequestedEventHandler)(unsafe.Pointer(retainNativeCallback[permissionCallback](e, e)))
	e.webResourceRequested = (*iCoreWebView2WebResourceRequestedEventHandler)(unsafe.Pointer(retainNativeCallback[resourceCallback](e, e)))
	e.acceleratorKeyPressed = (*ICoreWebView2AcceleratorKeyPressedEventHandler)(unsafe.Pointer(retainNativeCallback[acceleratorCallback](e, e)))
	e.navigationCompleted = (*ICoreWebView2NavigationCompletedEventHandler)(unsafe.Pointer(retainNativeCallback[navigationCallback](e, e)))
	e.processFailed = (*ICoreWebView2ProcessFailedEventHandler)(unsafe.Pointer(retainNativeCallback[processFailureCallback](e, e)))
	e.containsFullScreenElementChanged = (*ICoreWebView2ContainsFullScreenElementChangedEventHandler)(unsafe.Pointer(retainNativeCallback[fullscreenCallback](e, e)))
}

// Close the creator leases after controller Close and the retained COM interfaces.
// Native outstanding callbacks keep their own references. Clear Go ownership too,
// so closed wrappers cannot retain a Chromium -> wrapper -> Chromium cycle.
func (e *Chromium) releaseOwnedCallbacks() {
	owners := e.callbackLeases
	e.callbackLeases = nil
	e.envCompleted = nil
	e.controllerCompleted = nil
	e.webMessageReceived = nil
	e.permissionRequested = nil
	e.webResourceRequested = nil
	e.acceleratorKeyPressed = nil
	e.navigationCompleted = nil
	e.processFailed = nil
	e.containsFullScreenElementChanged = nil
	for _, owner := range owners {
		_ = owner.Close()
	}
}

// The Go loader retains this target through its own bridge, rather than a
// native callback allocation whose creator lease can close during cancellation.
func captureEnvironmentCallback(handler *iCoreWebView2CreateCoreWebView2EnvironmentCompletedHandler) environmentCompletedCallback {
	return combridge.Resolve[environmentCompletedCallback](uintptr(unsafe.Pointer(handler)))
}

func invokeEnvironmentCompleted(this uintptr, result uintptr, environment *ICoreWebView2Environment) uintptr {
	return combridge.Resolve[environmentCompletedCallback](this).EnvironmentCompleted(result, environment)
}
func invokeControllerCompleted(this uintptr, result uintptr, controller *ICoreWebView2Controller) uintptr {
	return combridge.Resolve[controllerCompletedCallback](this).CreateCoreWebView2ControllerCompleted(result, controller)
}
func liveNotificationTarget(this uintptr) *Chromium {
	target := combridge.Resolve[*Chromium](this)
	if target == nil || target.shuttingDown {
		return nil
	}
	return target
}
func invokeWebMessage(this uintptr, sender *ICoreWebView2, args *ICoreWebView2WebMessageReceivedEventArgs) uintptr {
	if target := liveNotificationTarget(this); target != nil {
		return target.MessageReceived(sender, args)
	}
	return 0
}
func invokePermission(this uintptr, sender *ICoreWebView2, args *iCoreWebView2PermissionRequestedEventArgs) uintptr {
	if target := liveNotificationTarget(this); target != nil {
		return target.PermissionRequested(sender, args)
	}
	return 0
}
func invokeResource(this uintptr, sender *ICoreWebView2, args *ICoreWebView2WebResourceRequestedEventArgs) uintptr {
	if target := liveNotificationTarget(this); target != nil {
		return target.WebResourceRequested(sender, args)
	}
	return 0
}
func invokeAccelerator(this uintptr, sender *ICoreWebView2Controller, args *ICoreWebView2AcceleratorKeyPressedEventArgs) uintptr {
	if target := liveNotificationTarget(this); target != nil {
		return target.AcceleratorKeyPressed(sender, args)
	}
	return 0
}
func invokeNavigation(this uintptr, sender *ICoreWebView2, args *ICoreWebView2NavigationCompletedEventArgs) uintptr {
	if target := liveNotificationTarget(this); target != nil {
		return target.NavigationCompleted(sender, args)
	}
	return 0
}
func invokeProcessFailure(this uintptr, sender *ICoreWebView2, args *ICoreWebView2ProcessFailedEventArgs) uintptr {
	if target := liveNotificationTarget(this); target != nil {
		return target.ProcessFailed(sender, args)
	}
	return 0
}
func invokeFullscreen(this uintptr, sender *ICoreWebView2, args *ICoreWebView2ContainsFullScreenElementChangedEventArgs) uintptr {
	if target := liveNotificationTarget(this); target != nil {
		return target.ContainsFullScreenElementChanged(sender, args)
	}
	return 0
}
