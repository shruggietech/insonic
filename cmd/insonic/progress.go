// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
)

func workProgressObserver(writer io.Writer, terminal, machine, quiet bool) func(catalog.Work) error {
	if !terminal || machine || quiet {
		return nil
	}
	return func(item catalog.Work) error {
		detail := ""
		var result map[string]json.RawMessage
		if json.Unmarshal(item.Result, &result) == nil {
			var items []json.RawMessage
			if json.Unmarshal(result["items"], &items) == nil && items != nil {
				detail = fmt.Sprintf(", %d observed items", len(items))
			}
			var files int
			if json.Unmarshal(result["files_verified"], &files) == nil && files > 0 {
				detail = fmt.Sprintf(", %d verified files", files)
			}
		}
		if _, err := fmt.Fprintf(writer, "Work %s: %s (%s)%s\n", safeText(item.ID), safeText(item.State), safeText(item.Phase), detail); err != nil {
			return contracts.Fail("unavailable")
		}
		if operatorContext.Debug {
			if _, err := fmt.Fprintf(writer, "Generation: %d; durable result bytes: %d\n", item.Generation, len(item.Result)); err != nil {
				return contracts.Fail("unavailable")
			}
		}
		return nil
	}
}
