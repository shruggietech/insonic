// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"strings"
	"time"
)

// OperationTimeout gives bounded artifact, media and model work enough time for
// selected storage and tools while retaining each caller's ordinary timeout.
func OperationTimeout(operation string, ordinary time.Duration) time.Duration {
	if strings.HasPrefix(operation, "artifacts.") || strings.HasPrefix(operation, "media.") || strings.HasPrefix(operation, "models.") {
		return 10 * time.Minute
	}
	return ordinary
}
