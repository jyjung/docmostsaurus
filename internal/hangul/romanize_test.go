package hangul

import "testing"

func TestRomanize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"basic vowels", "하이", "hai"},
		{"repeated", "하이하이", "haihai"},
		{"greeting", "안녕", "annyeong"},
		{"with final consonant", "값", "gabs"},
		{"english passthrough", "Google", "Google"},
		{"numbers passthrough", "123", "123"},
		{"mixed korean english", "Hello안녕", "Helloannyeong"},
		{"empty string", "", ""},
		{"space", "안녕 하세요", "annyeong haseyo"},
		{"double consonant initial", "빵", "ppang"},
		{"complex final", "닭", "dalg"},
		{"all initials sample", "가나다라마바사", "ganadaramabasa"},
		// 일본어 (그대로 통과 - SanitizeSpecialCharacters에서 처리)
		{"japanese hiragana", "こんにちは", "こんにちは"},
		{"japanese katakana", "カタカナ", "カタカナ"},
		{"japanese mixed", "東京タワー", "東京タワー"},
		// 중국어 (그대로 통과 - SanitizeSpecialCharacters에서 처리)
		{"chinese simplified", "你好", "你好"},
		{"chinese traditional", "謝謝", "謝謝"},
		// 혼합 테스트
		{"korean japanese mixed", "안녕こんにちは", "annyeongこんにちは"},
		{"korean chinese mixed", "안녕你好", "annyeong你好"},
		{"all languages mixed", "Hello안녕こんにちは你好123", "Helloannyeongこんにちは你好123"},
		// 특수문자 변환 테스트
		{"ampersand", "A & B", "A -and- B"},
		{"plus sign", "A + B", "A -plus- B"},
		{"hyphen passthrough", "A-B", "A-B"},
		{"parentheses removed", "A(B)", "AB"},
		{"brackets removed", "A[B]", "AB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Romanize(tt.input)
			if result != tt.expected {
				t.Errorf("Romanize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
