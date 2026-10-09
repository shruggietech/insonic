// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"path/filepath"
	"strconv"
)

func speakerModelCommand(args []string) bool {
	if len(args) < 2 || args[0] != "models" {
		return false
	}
	switch args[1] {
	case "dataset", "train", "speaker", "profile":
		return true
	}
	if args[1] == "list" {
		for _, arg := range args[2:] {
			if arg == "--speaker" {
				return true
			}
		}
	}
	return false
}

// parseSpeakerModels preserves base installation commands and requires exact
// UUIDs for trained output inspection and retrieval. Names never select latest.
func parseSpeakerModels(args []string) (string, string, json.RawMessage, error) {
	fail := func() (string, string, json.RawMessage, error) { return "", "", nil, contracts.Fail("invalid_request") }
	input := func(op, item string, index int, optional bool) (string, string, json.RawMessage, error) {
		if len(args) == index && optional {
			return op, item, nil, nil
		}
		if len(args) != index+2 || args[index] != "--input" {
			return fail()
		}
		raw, err := modelInput(args[index+1])
		return op, item, raw, err
	}
	if len(args) < 2 || args[0] != "models" {
		return fail()
	}
	if args[1] == "train" {
		return input("models.train", "", 2, false)
	}
	if args[1] == "list" {
		var speaker, after string
		limit := 100
		seen := map[string]bool{}
		for i := 2; i < len(args); i += 2 {
			if i+1 >= len(args) || seen[args[i]] {
				return fail()
			}
			seen[args[i]] = true
			switch args[i] {
			case "--speaker":
				speaker = args[i+1]
				if !contracts.ValidID(speaker) {
					return fail()
				}
			case "--after":
				after = args[i+1]
				if !contracts.ValidID(after) {
					return fail()
				}
			case "--limit":
				var err error
				limit, err = strconv.Atoi(args[i+1])
				if err != nil || limit < 1 || limit > 100 {
					return fail()
				}
			default:
				return fail()
			}
		}
		if speaker == "" {
			return fail()
		}
		raw, err := json.Marshal(map[string]any{"speaker_id": speaker, "after_id": after, "limit": limit})
		return "models.speaker.list", "", raw, err
	}
	if len(args) < 3 {
		return fail()
	}
	op := "models." + args[1] + "." + args[2]
	switch args[1] {
	case "dataset", "speaker":
		switch args[2] {
		case "list":
			return input(op, "", 3, true)
		case "create":
			if args[1] == "dataset" {
				return input(op, "", 3, false)
			}
		case "show":
			if len(args) == 4 && contracts.ValidID(args[3]) {
				return op, args[3], nil, nil
			}
		case "fetch":
			if args[1] == "speaker" && len(args) == 6 && contracts.ValidID(args[3]) && args[4] == "--destination" && args[5] != "" {
				destination, err := filepath.Abs(args[5])
				if err != nil {
					return fail()
				}
				raw, err := json.Marshal(map[string]string{"destination": destination})
				return op, args[3], raw, err
			}
		}
	case "profile":
		if args[2] == "set" && len(args) >= 4 && contracts.ValidID(args[3]) {
			return input(op, args[3], 4, false)
		}
	}
	return fail()
}
