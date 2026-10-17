// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/shruggietech/insonic/internal/setup"
	"os"
)

func main() { os.Exit(setup.Run(os.Args[1:])) }
