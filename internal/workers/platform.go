// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/process"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func glibcSufficient(observed, required string) bool {
	parse := func(value string) (int, int, bool) {
		parts := strings.Split(value, ".")
		if len(parts) < 2 {
			return 0, 0, false
		}
		major, e1 := strconv.Atoi(parts[0])
		minor, e2 := strconv.Atoi(parts[1])
		return major, minor, e1 == nil && e2 == nil && major >= 0 && minor >= 0
	}
	major, minor, ok := parse(observed)
	needMajor, needMinor, needOK := parse(required)
	return ok && needOK && (major > needMajor || major == needMajor && minor >= needMinor)
}

func (s Service) PlatformPrerequisite(ctx context.Context, p Profile) (string, error) {
	if runtime.GOOS != "linux" || p.MinimumGLibc == "" {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	run := s.Run
	if run == nil {
		run = process.Capture
	}
	result, e := run(ctx, process.Spec{Executable: "/usr/bin/getconf", Args: []string{"GNU_LIBC_VERSION"}, CleanEnv: true, Env: process.LocalEnvironment(), MaxOutput: 4096})
	parts := strings.Fields(string(result.Stdout))
	if e != nil || len(parts) != 2 || parts[0] != "glibc" {
		return "", &contracts.Error{Code: "unsupported_capability", Message: "The managed Linux profile requires GNU glibc 2.28 or newer, but this host could not verify it. Use a compatible host or explicitly reuse a verified environment."}
	}
	if !glibcSufficient(parts[1], p.MinimumGLibc) {
		return parts[1], &contracts.Error{Code: "incompatible_version", Message: "The installed GNU glibc is older than the managed Linux profile requires. Use glibc 2.28 or newer, or explicitly reuse a verified environment."}
	}
	return parts[1], nil
}
