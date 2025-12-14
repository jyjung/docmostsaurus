package hangul

import (
	"fmt"
	"strings"

	gohangul "github.com/suapapa/go_hangul"
)

var (
	// 초성 (0~18)
	initials = []string{"g", "kk", "n", "d", "tt", "r", "m", "b", "pp", "s", "ss", "", "j", "jj", "ch", "k", "t", "p", "h"}
	// 중성 (0~20)
	medials = []string{"a", "ae", "ya", "yae", "eo", "e", "yeo", "ye", "o", "wa", "wae", "oe", "yo", "u", "wo", "we", "wi", "yu", "eu", "ui", "i"}
	// 종성 (0~27, 0은 종성 없음)
	finals = []string{"", "k", "k", "ks", "n", "nj", "nh", "d", "l", "lg", "lm", "lb", "ls", "lt", "lp", "lh", "m", "b", "bs", "s", "ss", "ng", "j", "ch", "k", "t", "p", "h"}
)

// isASCIIAlphanumOrSpace checks if the rune is ASCII alphanumeric or space.
func isASCIIAlphanumOrSpace(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' '
}

// Romanize converts Korean text to romanized form.
func Romanize(word string) string {
	var result strings.Builder
	for _, r := range word {
		if !gohangul.IsHangul(r) {
			// 영어 알파벳, 숫자, 공백, 기본 ASCII는 그대로 통과
			if isASCIIAlphanumOrSpace(r) {
				result.WriteRune(r)
			} else {
				// 그 외 문자(일본어, 중국어 등)는 유니코드 코드포인트로 변환
				result.WriteString(fmt.Sprintf("U%04X", r))
			}
			continue
		}

		i, m, f := gohangul.Split(r)

		idxI := int(i - 0x1100)
		idxM := int(m - 0x1161)

		idxF := 0
		if f != 0 {
			idxF = int(f - 0x11A7)
		}

		if idxI >= 0 && idxI < len(initials) {
			result.WriteString(initials[idxI])
		}
		if idxM >= 0 && idxM < len(medials) {
			result.WriteString(medials[idxM])
		}
		if idxF > 0 && idxF < len(finals) {
			result.WriteString(finals[idxF])
		}
	}
	return result.String()
}
