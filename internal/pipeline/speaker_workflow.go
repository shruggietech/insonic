// SPDX-License-Identifier: Apache-2.0
package pipeline

import (
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/contracts"
	sd "github.com/shruggietech/insonic/internal/speakerdiscovery"
	"github.com/shruggietech/insonic/internal/voicemodels"
)

type SpeakerWorkflow struct {
	Provider      assistance.Config  `json:"provider"`
	Limits        sd.Limits          `json:"limits,omitempty"`
	Policy        sd.Policy          `json:"policy,omitempty"`
	Apply         bool               `json:"apply"`
	Enroll        bool               `json:"enroll"`
	SelectProfile bool               `json:"select_profile"`
	Recipe        voicemodels.Recipe `json:"recipe,omitempty"`
}

func NormalizeSpeakerWorkflow(v SpeakerWorkflow) (SpeakerWorkflow, error) {
	var e error
	v.Provider, e = assistance.Normalize(v.Provider)
	if e != nil || !v.Provider.Enabled {
		return v, contracts.Fail("invalid_request")
	}
	v.Limits, e = sd.NormalizeLimits(v.Limits)
	if e != nil {
		return v, e
	}
	v.Policy, e = sd.NormalizePolicy(v.Policy)
	if e != nil {
		return v, e
	}
	if v.Enroll && !v.Apply || v.SelectProfile && !v.Enroll {
		return v, contracts.Fail("invalid_request")
	}
	if e = voicemodels.ValidateRecipe(v.Recipe); e != nil {
		return v, e
	}
	return v, nil
}
