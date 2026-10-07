//go:build darwin && cgo

// SPDX-License-Identifier: Apache-2.0
package main

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <stdlib.h>
#include <string.h>

static SecKeychainRef original_keychain = NULL;
static SecKeychainRef fixture_keychain = NULL;
static OSStatus fixture_start(const char *path, const void *password, UInt32 length) {
    OSStatus status = SecKeychainSetUserInteractionAllowed(false);
    if (status != errSecSuccess) return status;
    status = SecKeychainCopyDefault(&original_keychain);
    if (status != errSecSuccess) return status;
    status = SecKeychainCreate(path, length, password, false, NULL, &fixture_keychain);
    if (status != errSecSuccess) return status;
    status = SecKeychainUnlock(fixture_keychain, length, password, true);
    if (status != errSecSuccess) return status;
    return SecKeychainSetDefault(fixture_keychain);
}
static OSStatus fixture_finish(void) {
    OSStatus status = errSecSuccess;
    if (original_keychain) { status = SecKeychainSetDefault(original_keychain); CFRelease(original_keychain); original_keychain = NULL; }
    if (fixture_keychain) { OSStatus deletion = SecKeychainDelete(fixture_keychain); if (status == errSecSuccess) status = deletion; CFRelease(fixture_keychain); fixture_keychain = NULL; }
    return status;
}
*/
import "C"
import (
	"crypto/rand"
	"github.com/shruggietech/insonic/internal/contracts"
	"path/filepath"
	"unsafe"
)

func nativeFixture(directory string) (func() error, error) {
	path := C.CString(filepath.Join(directory, "native-fixture.keychain"))
	defer C.free(unsafe.Pointer(path))
	password := make([]byte, 32)
	if _, e := rand.Read(password); e != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer clear(password)
	secret := C.CBytes(password)
	defer func() { C.memset(secret, 0, C.size_t(len(password))); C.free(secret) }()
	if C.fixture_start(path, secret, C.UInt32(len(password))) != C.errSecSuccess {
		C.fixture_finish()
		return nil, contracts.Fail("unavailable")
	}
	return func() error {
		if C.fixture_finish() != C.errSecSuccess {
			return contracts.Fail("unavailable")
		}
		return nil
	}, nil
}
