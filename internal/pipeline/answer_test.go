// SPDX-License-Identifier: Apache-2.0
package pipeline

import (
	"github.com/shruggietech/insonic/internal/assistance"
	"testing"
)

func TestAnswerOnlyPipelineElection(t *testing.T) {
	c := assistance.DefaultConfig()
	c.Enabled = true
	c.Endpoint = "http://127.0.0.1:1"
	c.Model = "fixture"
	d := Definition{AnswerSynthesis: &assistance.SynthesisConfig{Provider: c}}
	if err := Validate("local", d); err != nil {
		t.Fatal(err)
	}
	if err := Validate("connected", d); err == nil {
		t.Fatal("local route accepted as connected")
	}
	c.Route = "hosted"
	d.AnswerSynthesis = &assistance.SynthesisConfig{Provider: c}
	if err := Validate("connected", d); err != nil {
		t.Fatal(err)
	}
	if err := Validate("local", d); err == nil {
		t.Fatal("hosted route accepted as local")
	}
	if err := Validate("custom", d); err != nil {
		t.Fatal(err)
	}
	two := 2
	d.AnswerSynthesis.MaxRepairs = &two
	if err := Validate("custom", d); err == nil {
		t.Fatal("unbounded repair accepted")
	}
}
