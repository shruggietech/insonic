//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"fmt"
	"time"
)

const defaultNativeWindowJoinSeconds = 2

type nativeWindowJoinKey struct{}

func selectedNativeWindowJoinSeconds(value string) (int, error) {
	switch value {
	case "":
		return defaultNativeWindowJoinSeconds, nil
	case "2":
		return 2, nil
	case "10":
		return 10, nil
	default:
		return 0, fmt.Errorf("native qualification join allowance must be 2 or 10 seconds")
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
