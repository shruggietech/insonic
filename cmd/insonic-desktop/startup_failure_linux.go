//go:build desktop && linux

// SPDX-License-Identifier: Apache-2.0
package main

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>
#include <stdlib.h>

static int insonic_startup_error(const char *message) {
    if (!gtk_init_check(NULL, NULL)) return 0;
    GtkWidget *dialog = gtk_message_dialog_new(NULL, GTK_DIALOG_MODAL,
        GTK_MESSAGE_ERROR, GTK_BUTTONS_CLOSE, "%s", message);
    if (!dialog) return 0;
    gtk_window_set_title(GTK_WINDOW(dialog), "insonic could not start");
    gtk_dialog_run(GTK_DIALOG(dialog));
    gtk_widget_destroy(dialog);
    return 1;
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

func presentStartupFailure(message string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	text := C.CString(message)
	defer C.free(unsafe.Pointer(text))
	if C.insonic_startup_error(text) == 0 {
		return fmt.Errorf("native startup feedback is unavailable")
	}
	return nil
}
