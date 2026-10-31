// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/library"
)

func TestWorkerBootstrapKeepsElectedPathsAsLiteralOperands(t *testing.T) {
	worker := `C:\private folder\worker ' $;.py`
	receipt := library.PinnedFile{Path: `C:\private folder\receipt ' $;.json`, SHA256: strings.Repeat("a", 64)}
	args := WorkerArgs(worker, receipt)
	if len(args) != 7 || args[0] != "-I" || args[1] != "-B" || args[2] != "-c" || args[4] != worker || args[5] != receipt.Path || args[6] != receipt.SHA256 {
		t.Fatalf("invalid bootstrap operands: %#v", args)
	}
	if strings.Contains(args[3], worker) || strings.Contains(args[3], receipt.Path) || !strings.Contains(args[3], "sys.modules['insonic_native']") || !strings.Contains(args[3], "_native.set_receipt(sys.argv[2],sys.argv[3])") {
		t.Fatal("bootstrap interpolates paths or omits exact native election")
	}
	legacy := WorkerArgs(worker)
	if legacy[5] != "" || legacy[6] != "" {
		t.Fatal("advanced legacy worker acquired a fabricated receipt")
	}
}
