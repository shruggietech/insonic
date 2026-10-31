// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"encoding/json"

	"github.com/shruggietech/insonic/internal/library"
)

// NativeBootstrap installs only the binary's embedded helper, never Python's
// ambient module search path or an unpinned adjacent script.
func NativeBootstrap() string {
	source, _ := assets.ReadFile("native.py")
	encoded, _ := json.Marshal(string(source))
	return "import sys,types\n_native=types.ModuleType('insonic_native')\nexec(compile(" + string(encoded) + ", '<insonic-native>', 'exec'), _native.__dict__)\nsys.modules['insonic_native']=_native\n"
}

// WorkerArgs supplies a verified receipt election to each concrete child attempt.
func WorkerArgs(workerPath string, receipt ...library.PinnedFile) []string {
	pin := library.PinnedFile{}
	if len(receipt) > 0 {
		pin = receipt[0]
	}
	code := NativeBootstrap() + "import json,runpy\ntry:\n _native.set_receipt(sys.argv[2],sys.argv[3])\nexcept _native.NativeFailure as error:\n sys.stderr.write(json.dumps({'error':str(error)})+'\\n')\n sys.exit(1)\nrunpy.run_path(sys.argv[1],run_name='__main__')\n"
	return []string{"-I", "-B", "-c", code, workerPath, pin.Path, pin.SHA256}
}
