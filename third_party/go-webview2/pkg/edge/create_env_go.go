//go:build windows && !native_webview2loader

package edge

import (
	"unsafe"

	"github.com/wailsapp/go-webview2/webviewloader"
)

func createCoreWebView2EnvironmentWithOptions(browserExecutableFolder, userDataFolder string, environmentCompletedHandle *iCoreWebView2CreateCoreWebView2EnvironmentCompletedHandler, additionalBrowserArgs string) error {
	// Capture the Go target while its creator lease is live. The asynchronous
	// loader owns this adapter even if native-window cleanup closes our lease.
	e := captureEnvironmentHandler(environmentCompletedHandle)
	return webviewloader.CreateCoreWebView2EnvironmentWithOptions(
		e,
		webviewloader.WithBrowserExecutableFolder(browserExecutableFolder),
		webviewloader.WithUserDataFolder(userDataFolder),
		webviewloader.WithAdditionalBrowserArguments(additionalBrowserArgs),
	)
}

func captureEnvironmentHandler(handler *iCoreWebView2CreateCoreWebView2EnvironmentCompletedHandler) *environmentCreatedHandler {
	return &environmentCreatedHandler{captureEnvironmentCallback(handler)}
}

type environmentCreatedHandler struct {
	originalHandler environmentCompletedCallback
}

func (r *environmentCreatedHandler) EnvironmentCompleted(errorCode webviewloader.HRESULT, createdEnvironment *webviewloader.ICoreWebView2Environment) webviewloader.HRESULT {
	env := (*ICoreWebView2Environment)(unsafe.Pointer(createdEnvironment))
	res := r.originalHandler.EnvironmentCompleted(uintptr(errorCode), env)
	return webviewloader.HRESULT(res)
}
