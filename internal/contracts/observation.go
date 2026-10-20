// SPDX-License-Identifier: Apache-2.0
package contracts

import "reflect"

// ObservedResult removes an unestablished value result from a failed operation.
// Handlers must call it before converting a value struct to a public view. A
// nonzero partial observation, and scalar false/zero results, remain meaningful.
func ObservedResult(result any, err error) any {
	if err == nil || result == nil {
		return result
	}
	value := reflect.ValueOf(result)
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Struct:
		if value.IsZero() {
			return nil
		}
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
		if value.IsNil() {
			return nil
		}
	}
	return result
}
