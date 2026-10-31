// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"strings"

	"github.com/shruggietech/insonic/internal/contracts"
)

// Supported Whisper tokenizer languages for the pinned faster-whisper adapter.
// Hosted adapters retain their own language contracts.
const whisperLanguages = "af am ar as az ba be bg bn bo br bs ca cs cy da de el en es et eu fa fi fo fr gl gu ha haw he hi hr ht hu hy id is it ja jw ka kk km kn ko la lb ln lo lt lv mg mi mk ml mn mr ms mt my ne nl nn no oc pa pl ps pt ro ru sa sd si sk sl sn so sq sr su sv sw ta te tg th tk tl tr tt uk ur uz vi yi yo yue zh"

func ValidateLocalLanguage(language string) error {
	if language == "" || language == "auto" || len(language) >= 2 && len(language) <= 3 && strings.Contains(" "+whisperLanguages+" ", " "+language+" ") {
		return nil
	}
	return contracts.InputFailure("unsupported_language", contracts.InputViolation{Pointer: "/recognition/language", Expected: "string"})
}
