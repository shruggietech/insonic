// SPDX-License-Identifier: Apache-2.0
package assistance

import "github.com/shruggietech/insonic/internal/contracts"

type SynthesisConfig struct {
	Provider   Config `json:"provider"`
	MaxRepairs *int   `json:"max_repairs,omitempty"`
}

func NormalizeSynthesis(c SynthesisConfig) (SynthesisConfig, error) {
	var err error
	c.Provider, err = Normalize(c.Provider)
	if err != nil {
		return c, err
	}
	if !c.Provider.Enabled {
		return c, contracts.Fail("unavailable")
	}
	if c.MaxRepairs == nil {
		one := 1
		c.MaxRepairs = &one
	}
	if *c.MaxRepairs < 0 || *c.MaxRepairs > 1 {
		return c, contracts.Fail("invalid_request")
	}
	return c, nil
}
