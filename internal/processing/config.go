// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
)

func applyExecutionLimits(config Config, path string, limits *ExecutionLimits) (Config, error) {
	if limits == nil {
		return config, nil
	}
	if limits.MaxAudioBytes < 1 || limits.MaxAudioBytes > 64<<30 || limits.MaxResponseBytes < 1 || limits.MaxResponseBytes > 16<<20 || limits.TimeoutMS < 1 || limits.TimeoutMS > 86400000 {
		return config, contracts.Fail("invalid_request")
	}
	info, e := os.Stat(path)
	if e != nil || !info.Mode().IsRegular() {
		return config, contracts.Fail("invalid_audio")
	}
	if info.Size() > limits.MaxAudioBytes {
		return config, contracts.Fail("input_limit")
	}
	// The elected stage can lower global safety bounds but cannot exceed them.
	if limits.MaxResponseBytes < int64(config.MaxOutputBytes) {
		config.MaxOutputBytes = int(limits.MaxResponseBytes)
	}
	if limits.TimeoutMS < config.TimeoutMS {
		config.TimeoutMS = limits.TimeoutMS
	}
	return config, nil
}

var pinnedDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)

// ValidateConfig checks settings without opening executable or model files.
// Omitted inference tools remain valid for operations that do not elect them.
func ValidateConfig(c Config) error {
	if _, err := defaults(c); err != nil {
		return err
	}
	validPin := func(file library.PinnedFile) bool {
		return filepath.IsAbs(file.Path) && pinnedDigest.MatchString(file.SHA256)
	}
	for _, file := range []library.PinnedFile{c.RecognitionPython, c.DiarizationPython, c.Worker} {
		if file.Path == "" && file.SHA256 == "" {
			continue
		}
		if !validPin(file) {
			return contracts.Fail("invalid_request")
		}
	}
	tool := c.FFmpeg
	if tool.Path == "" && tool.SHA256 == "" && tool.Version == "" && len(tool.SupportFiles) == 0 && tool.Interpreter == nil {
		return nil
	}
	if !validPin(library.PinnedFile{Path: tool.Path, SHA256: tool.SHA256}) || tool.Version == "" {
		return contracts.Fail("invalid_request")
	}
	for _, file := range tool.SupportFiles {
		if !validPin(file) {
			return contracts.Fail("invalid_request")
		}
	}
	if tool.Interpreter != nil && !validPin(*tool.Interpreter) {
		return contracts.Fail("invalid_request")
	}
	return nil
}
