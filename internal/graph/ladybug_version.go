//go:build system_ladybug

// SPDX-License-Identifier: Apache-2.0
package graph

// #include <lbug.h>
import "C"

func ladybugVersion() string {
	v := C.lbug_get_version()
	defer C.lbug_destroy_string(v)
	return C.GoString(v)
}
