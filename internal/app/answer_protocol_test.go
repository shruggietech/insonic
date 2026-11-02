// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestAnswerTransportProtocolRepairIsBounded(t *testing.T) {
	a, p, c := answerSynthesisApp(t)
	calls := 0
	a.AnswerFixture = func(ctx context.Context, in assistance.Request, config assistance.Config) (assistance.Generated, error) {
		calls++
		if calls == 1 {
			return assistance.Generated{}, contracts.Condition("invalid_request", "provider_protocol")
		}
		e, err := fixtureAnswer(in)
		raw, _ := json.Marshal(e)
		return assistance.Generated{Raw: raw, ObservedRoute: "local"}, err
	}
	got, err := a.synthesizeAnswer(context.Background(), p, c, nil)
	if err != nil || calls != 2 || len(got.Attempts) != 2 || got.Attempts[0].Outcome != "rejected" || got.Attempts[1].Outcome != "validated" {
		t.Fatal(got, calls, err)
	}
	zero := 0
	c.MaxRepairs = &zero
	calls = 0
	got, err = a.synthesizeAnswer(context.Background(), p, c, nil)
	if err != nil || calls != 1 || got.Answer.State != "insufficient" {
		t.Fatal(got, calls, err)
	}
}
