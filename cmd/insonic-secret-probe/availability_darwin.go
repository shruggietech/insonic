//go:build cgo

package main

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <stdlib.h>
#include <string.h>

static OSStatus insonicMissingLookup(SecKeychainRef keychain, const char *service, const char *account) {
    SecKeychainStatus state = 0;
    OSStatus status = SecKeychainGetStatus(keychain, &state);
    if (status != errSecSuccess) return status;
    if (!(state & kSecUnlockStateStatus) || !(state & kSecReadPermStatus))
        return errSecInteractionNotAllowed;
    return SecKeychainFindGenericPassword(keychain, (UInt32)strlen(service), service,
        (UInt32)strlen(account), account, NULL, NULL, NULL);
}

static OSStatus insonicDefaultLookup(const char *service, const char *account) {
    if (SecKeychainSetUserInteractionAllowed(false) != errSecSuccess)
        return errSecInteractionNotAllowed;
    SecKeychainRef keychain = NULL;
    OSStatus status = SecKeychainCopyDefault(&keychain);
    if (status == errSecSuccess) {
        status = insonicMissingLookup(keychain, service, account);
        CFRelease(keychain);
    }
    return status;
}

static int insonicLockedFixture(const char *path) {
    SecKeychainSetUserInteractionAllowed(false);
    const char *password = "fixture-only-password";
    SecKeychainRef keychain = NULL;
    if (SecKeychainCreate(path, (UInt32)strlen(password), password, false, NULL, &keychain) != errSecSuccess)
        return 0;
    int passed = 0;
    if (SecKeychainUnlock(keychain, (UInt32)strlen(password), password, true) == errSecSuccess &&
        insonicMissingLookup(keychain, "insonic-empty-fixture", "missing") == errSecItemNotFound &&
        SecKeychainLock(keychain) == errSecSuccess &&
        insonicMissingLookup(keychain, "insonic-empty-fixture", "missing") == errSecInteractionNotAllowed)
        passed = 1;
    if (SecKeychainDelete(keychain) != errSecSuccess) passed = 0;
    CFRelease(keychain);
    return passed;
}
*/
import "C"

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"unsafe"
)

func available() bool {
	service := C.CString("insonic-qualification-" + contracts.ID())
	account := C.CString(contracts.ID())
	defer C.free(unsafe.Pointer(service))
	defer C.free(unsafe.Pointer(account))
	return C.insonicDefaultLookup(service, account) == C.errSecItemNotFound
}

func lockedFixture(path string) bool {
	value := C.CString(path)
	defer C.free(unsafe.Pointer(value))
	return C.insonicLockedFixture(value) == 1
}
