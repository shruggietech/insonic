// SPDX-License-Identifier: Apache-2.0
package app

import (
	"io"
	"os"
	"path/filepath"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/packageenv"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/workspace"
)

func ReadMediaTools(w *workspace.Workspace) (library.Tools, error) {
	var tools library.Tools
	file, err := os.Open(filepath.Join(w.Control, "media-tools.json"))
	if os.IsNotExist(err) {
		defaults, found, err := packageenv.Defaults()
		if err != nil {
			return tools, err
		}
		if found {
			return defaults.Media, nil
		}
		return tools, nil
	}
	if err != nil {
		return tools, contracts.Fail("unavailable")
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 || strictPayload(raw, &tools) != nil {
		return library.Tools{}, contracts.Fail("invalid_request")
	}
	return tools, nil
}

func ReadProcessingTools(w *workspace.Workspace) (ProcessingTools, error) {
	var tools ProcessingTools
	path := filepath.Join(w.Control, "processing-tools.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		defaults, found, err := packageenv.Defaults()
		if err != nil {
			return tools, err
		}
		if !found {
			return tools, contracts.Fail("unavailable")
		}
		tools = ProcessingTools{Kind: "processing-tools", Version: contracts.Version, Cueson: defaults.Cueson,
			Processing: processing.Config{FFmpeg: defaults.Media.FFmpeg}}
	} else if err := readBoundedConfiguration(path, &tools); err != nil {
		return tools, err
	}
	if err := ValidateProcessingTools(tools); err != nil {
		return ProcessingTools{}, err
	}
	return tools, nil
}
