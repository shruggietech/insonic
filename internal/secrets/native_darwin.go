//go:build darwin && cgo

// SPDX-License-Identifier: Apache-2.0
package secrets

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <stdlib.h>
#include <string.h>

static OSStatus insonic_keychain(SecKeychainRef *keychain) {
    OSStatus status = SecKeychainSetUserInteractionAllowed(false);
    if (status != errSecSuccess) return status;
    status = SecKeychainCopyDefault(keychain);
    if (status != errSecSuccess) return status;
    SecKeychainStatus state = 0;
    status = SecKeychainGetStatus(*keychain, &state);
    if (status != errSecSuccess || !(state & kSecUnlockStateStatus) || !(state & kSecReadPermStatus) || !(state & kSecWritePermStatus)) {
        CFRelease(*keychain); *keychain = NULL;
        return errSecInteractionNotAllowed;
    }
    return errSecSuccess;
}
static OSStatus insonic_available(void) {
    SecKeychainRef keychain = NULL;
    OSStatus status = insonic_keychain(&keychain);
    if (keychain) CFRelease(keychain);
    return status;
}
static OSStatus insonic_get(const char *service, const char *account, UInt32 *length, void **data) {
    SecKeychainRef keychain = NULL;
    OSStatus status = insonic_keychain(&keychain);
    if (status == errSecSuccess) {
        status = SecKeychainFindGenericPassword(keychain, (UInt32)strlen(service), service, (UInt32)strlen(account), account, length, data, NULL);
        CFRelease(keychain);
    }
    return status;
}
static OSStatus insonic_put(const char *service, const char *account, UInt32 length, const void *data) {
    SecKeychainRef keychain = NULL;
    OSStatus status = insonic_keychain(&keychain);
    if (status != errSecSuccess) return status;
    SecKeychainItemRef item = NULL;
    status = SecKeychainFindGenericPassword(keychain, (UInt32)strlen(service), service, (UInt32)strlen(account), account, NULL, NULL, &item);
    if (status == errSecItemNotFound)
        status = SecKeychainAddGenericPassword(keychain, (UInt32)strlen(service), service, (UInt32)strlen(account), account, length, data, NULL);
    else if (status == errSecSuccess) {
        status = SecKeychainItemModifyAttributesAndData(item, NULL, length, data);
        CFRelease(item);
    }
    CFRelease(keychain);
    return status;
}
static OSStatus insonic_delete(const char *service, const char *account) {
    SecKeychainRef keychain = NULL;
    OSStatus status = insonic_keychain(&keychain);
    if (status != errSecSuccess) return status;
    SecKeychainItemRef item = NULL;
    status = SecKeychainFindGenericPassword(keychain, (UInt32)strlen(service), service, (UInt32)strlen(account), account, NULL, NULL, &item);
    if (status == errSecSuccess) { status = SecKeychainItemDelete(item); CFRelease(item); }
    else if (status == errSecItemNotFound) status = errSecSuccess;
    CFRelease(keychain);
    return status;
}
*/
import "C"
import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"unsafe"
)

type darwinNative struct{ service string }

func newNative(service string) nativeBackend { return darwinNative{service} }
func (n darwinNative) available(ctx context.Context) error {
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	if C.insonic_available() != C.errSecSuccess {
		return contracts.Fail("unavailable")
	}
	return nil
}
func (n darwinNative) get(ctx context.Context, id string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	service, account := C.CString(n.service), C.CString(id)
	defer C.free(unsafe.Pointer(service))
	defer C.free(unsafe.Pointer(account))
	var length C.UInt32
	var data unsafe.Pointer
	status := C.insonic_get(service, account, &length, &data)
	if status == C.errSecItemNotFound {
		return nil, contracts.Fail("not_found")
	}
	if status != C.errSecSuccess {
		return nil, contracts.Fail("unavailable")
	}
	defer func() { C.memset(data, 0, C.size_t(length)); C.SecKeychainItemFreeContent(nil, data) }()
	if length > MaxValueBytes {
		return nil, contracts.Fail("unavailable")
	}
	return C.GoBytes(data, C.int(length)), nil
}
func (n darwinNative) put(ctx context.Context, id string, value []byte) error {
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	service, account := C.CString(n.service), C.CString(id)
	defer C.free(unsafe.Pointer(service))
	defer C.free(unsafe.Pointer(account))
	data := C.CBytes(value)
	defer func() { C.memset(data, 0, C.size_t(len(value))); C.free(data) }()
	if C.insonic_put(service, account, C.UInt32(len(value)), data) != C.errSecSuccess {
		return contracts.Fail("unavailable")
	}
	return nil
}
func (n darwinNative) remove(ctx context.Context, id string) error {
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	service, account := C.CString(n.service), C.CString(id)
	defer C.free(unsafe.Pointer(service))
	defer C.free(unsafe.Pointer(account))
	if C.insonic_delete(service, account) != C.errSecSuccess {
		return contracts.Fail("unavailable")
	}
	return nil
}
