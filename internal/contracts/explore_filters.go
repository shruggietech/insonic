// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"encoding/json"
)

func hasNormalizedFilters(f QueryFilter) bool { raw, _ := json.Marshal(f); return string(raw) != "{}" }
