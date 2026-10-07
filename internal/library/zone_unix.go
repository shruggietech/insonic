// SPDX-License-Identifier: Apache-2.0
//go:build !windows

package library

import (
	"os"
	"strings"
)

func platformLocalZone() (string, error) {
	if value := strings.TrimPrefix(os.Getenv("TZ"), ":"); value != "" {
		if _, e := loadZone(value); e == nil {
			return value, nil
		}
	}
	if path, e := os.Readlink("/etc/localtime"); e == nil {
		if index := strings.Index(path, "zoneinfo/"); index >= 0 {
			value := path[index+len("zoneinfo/"):]
			if _, e := loadZone(value); e == nil {
				return value, nil
			}
		}
	}
	if data, e := os.ReadFile("/etc/timezone"); e == nil {
		value := strings.TrimSpace(string(data))
		if _, e := loadZone(value); e == nil {
			return value, nil
		}
	}
	return "", &zoneError{}
}
