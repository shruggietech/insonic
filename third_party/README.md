# Maintained WebView binding

The Windows desktop uses the pinned `github.com/wailsapp/go-webview2` v1.0.22 source with an explicit owned-window shutdown patch. Wails destroys its window without closing the controller. The added window subclass closes that controller once on its owning UI thread and preserves ordinary native destruction. A close error survives the remaining application cleanup and fails qualification.

The source manifest records every original file hash, the original Go module checksum, every maintained file hash and the maintained tree checksum. Only `pkg/edge/chromium.go` changes upstream bytes. The additions contain the lifetime implementation, an unshown native-window regression, the project license and an explicit patch notice. Original upstream license bytes remain intact.

Repository checks verify all source bytes, membership, the upstream `go.sum` binding, the declared replacement and the precise patch scope. Package notices independently rehash the maintained source before copying its notices. Go records this exact relative replacement without a module checksum, so the clean product source revision and the verified maintained-source manifest provide its source authority. Other local replacements remain rejected. Installed notice verification requires the explicit patch notice and both applicable licenses.

Updates require a reviewed change to the pinned upstream identity, source manifest and verifier allowlist. Product versions and existing immutable release records are independent of this source-maintenance record. Local SDK verification selects only the message-only-window lifetime test; actual browser and product qualification runs on disposable CI.
