// SPDX-License-Identifier: Apache-2.0
//go:build windows

package library

import "golang.org/x/sys/windows/registry"

func platformLocalZone() (string, error) {
	key, e := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\TimeZoneInformation`, registry.QUERY_VALUE)
	if e != nil {
		return "", &zoneError{}
	}
	defer key.Close()
	name, _, e := key.GetStringValue("TimeZoneKeyName")
	if e != nil {
		return "", &zoneError{}
	}
	if zone, ok := windowsMapping(name); ok {
		return zone, nil
	}
	return "", &zoneError{}
}
