// SPDX-License-Identifier: Apache-2.0
package runtime

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"time"
)

// OperationTimeout gives bounded configuration, artifact, media and model work enough time for
// selected storage and tools while retaining each caller's ordinary timeout.
func OperationTimeout(operation string, ordinary time.Duration) time.Duration {
	if operation == "query.assist" {
		return 90 * time.Second
	}
	if contracts.ConfigurationOperation(operation) || contracts.DesktopOperation(operation) || strings.HasPrefix(operation, "recordings.") || strings.HasPrefix(operation, "artifacts.") || strings.HasPrefix(operation, "media.") || strings.HasPrefix(operation, "models.") {
		return 10 * time.Minute
	}
	return ordinary
}
