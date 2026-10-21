//go:build production || dev

// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/wailsapp/wails/v2/internal/frontend"
)

type lifecycleFrontend struct {
	frontend.Frontend
	trace   *[]string
	failure error
}

func (f *lifecycleFrontend) Run(context.Context) error {
	*f.trace = append(*f.trace, "run")
	return f.failure
}
func (f *lifecycleFrontend) RunMainLoop() { *f.trace = append(*f.trace, "loop") }
func (f *lifecycleFrontend) WindowClose() { *f.trace = append(*f.trace, "close") }
func TestStartupFailureReturnsBeforeMainLoopAndClosesFrontend(t *testing.T) {
	cause := errors.New("injected frontend startup failure")
	var trace []string
	app := &App{ctx: context.Background(), frontend: &lifecycleFrontend{trace: &trace, failure: cause}, shutdownCallback: func(context.Context) { trace = append(trace, "shutdown") }}
	if err := app.Run(); !errors.Is(err, cause) {
		t.Fatal("original startup cause lost", err)
	}
	if !reflect.DeepEqual(trace, []string{"run", "close", "shutdown"}) {
		t.Fatal("failed startup entered ordinary main loop or skipped cleanup", trace)
	}
}
func TestStartupSuccessKeepsMainLoopAndShutdownOrder(t *testing.T) {
	var trace []string
	app := &App{ctx: context.Background(), frontend: &lifecycleFrontend{trace: &trace}, shutdownCallback: func(context.Context) { trace = append(trace, "shutdown") }}
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(trace, []string{"run", "loop", "close", "shutdown"}) {
		t.Fatal("successful lifecycle changed", trace)
	}
}
func TestFreshFrontendLifecycleWorksAfterReturnedFailure(t *testing.T) {
	cause := errors.New("first startup failure")
	var failed, valid []string
	first := &App{ctx: context.Background(), frontend: &lifecycleFrontend{trace: &failed, failure: cause}}
	if !errors.Is(first.Run(), cause) {
		t.Fatal("first startup failure lost")
	}
	next := &App{ctx: context.Background(), frontend: &lifecycleFrontend{trace: &valid}}
	if err := next.Run(); err != nil || !reflect.DeepEqual(valid, []string{"run", "loop", "close"}) {
		t.Fatal("fresh frontend lifecycle failed", err, valid)
	}
}
