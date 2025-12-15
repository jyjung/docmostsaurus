package hangul

import (
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

	// specialCharReplacements maps special characters to their safe replacements
	specialCharReplacements = map[rune]string{
		'&': "-and-",
		'+': "-plus-",
		'@': "-at-",
		'#': "-num-",
		'%': "-pct-",
		'=': "-eq-",
		'-': "-",
		'_': "_",
		'.': ".",
		'/': "/",
	}

	// charactersToRemove are special characters that should be removed entirely
	charactersToRemove = map[rune]bool{
		'(':  true,
		')':  true,
		'[':  true,
		']':  true,
		'{':  true,
		'}':  true,
		'\'': true,
		'"':  true,
		',':  true,
		';':  true,
		'!':  true,
		'$':  true,
		'^':  true,
		'`':  true,
		'~':  true,
		'<':  true,
		'>':  true,
		'|':  true,
		'*':  true,
		'?':  true,
	}
)

// isASCIIAlphanumOrSpace checks if the rune is ASCII alphanumeric or space.
func isASCIIAlphanumOrSpace(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' '
}

// Romanize converts Korean text to romanized form.
// It also handles special characters consistently with SanitizeSpecialCharacters.
func Romanize(word string) string {
	var result strings.Builder
	for _, r := range word {
		if !gohangul.IsHangul(r) {
			// 영어 알파벳, 숫자, 공백은 그대로 통과
			if isASCIIAlphanumOrSpace(r) {
				result.WriteRune(r)
				continue
			}

			// Check if it's a character to remove
			if charactersToRemove[r] {
				continue
			}

			// Check if it has a replacement
			if replacement, ok := specialCharReplacements[r]; ok {
				result.WriteString(replacement)
				continue
			}

			// 그 외 문자(일본어, 중국어 등)는 그냥 통과 (유니코드 변환하지 않음)
			// Docusaurus에서 지원되지 않는 문자는 SanitizeSpecialCharacters에서 처리됨
			result.WriteRune(r)
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
