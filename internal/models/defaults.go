// SPDX-License-Identifier: Apache-2.0
package models

import (
	_ "embed"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
)

//go:embed defaults.json
var defaultBytes []byte

type Choice struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Role      string   `json:"role"`
	ModelCard string   `json:"model_card"`
	Access    string   `json:"access"`
	Manifest  Manifest `json:"manifest"`
}

func Defaults() []Choice { var c []Choice; json.Unmarshal(defaultBytes, &c); return c }
func (c Choice) Bytes() int64 {
	var n int64
	for _, f := range c.Manifest.Files {
		n += f.Size
	}
	return n
}
func Default(id, credential string) (Choice, error) {
	if credential != "" && !contracts.ValidID(credential) {
		return Choice{}, contracts.Fail("invalid_request")
	}
	for _, c := range Defaults() {
		if c.ID == id {
			for i := range c.Manifest.Files {
				c.Manifest.Files[i].CredentialID = credential
			}
			return c, nil
		}
	}
	return Choice{}, contracts.Fail("not_found")
}
