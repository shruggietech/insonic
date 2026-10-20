//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package process

import (
	"context"
	"fmt"
)

func waitDescendants(context.Context, *processTree) (DescendantReport, error) {
	return DescendantReport{}, fmt.Errorf("owned job release observation requires Windows")
}
