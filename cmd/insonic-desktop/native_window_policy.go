//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"fmt"
	"github.com/shruggietech/insonic/internal/qualificationtiming"
	"strconv"
	"time"
)

// Qualification policy: sixty seconds per actual UI completion, bounded by the
// existing watchdog and enclosing child/matrix deadlines. This is not a sleep.
var defaultNativeWindowJoinSeconds = qualificationtiming.Seconds("native_ui_seconds")

type nativeWindowJoinKey struct{}

func selectedNativeWindowJoinSeconds(value string) (int, error) {
	switch value {
	case "":
		return defaultNativeWindowJoinSeconds, nil
	case "2":
		return 2, nil
	case "10":
		return 10, nil
	case strconv.Itoa(defaultNativeWindowJoinSeconds):
		return defaultNativeWindowJoinSeconds, nil
	default:
		return 0, fmt.Errorf("native qualification join allowance must be 2, 10 or %d seconds", defaultNativeWindowJoinSeconds)
	}
}

func nativeWindowJoinContext(ctx context.Context, seconds int) context.Context {
	if seconds == 0 {
		seconds = defaultNativeWindowJoinSeconds
	}
	return context.WithValue(ctx, nativeWindowJoinKey{}, seconds)
}

func nativeWindowJoinBudget(ctx context.Context) time.Duration {
	seconds, ok := ctx.Value(nativeWindowJoinKey{}).(int)
	if !ok {
		seconds = defaultNativeWindowJoinSeconds
	}
	return time.Duration(seconds) * time.Second
}
