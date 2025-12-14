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
		// 일본어 (유니코드 코드포인트로 변환)
		{"japanese hiragana", "こんにちは", "U3053U3093U306BU3061U306F"},
		{"japanese katakana", "カタカナ", "U30ABU30BFU30ABU30CA"},
		{"japanese mixed", "東京タワー", "U6771U4EACU30BFU30EFU30FC"},
		// 중국어 (유니코드 코드포인트로 변환)
		{"chinese simplified", "你好", "U4F60U597D"},
		{"chinese traditional", "謝謝", "U8B1DU8B1D"},
		// 혼합 테스트
		{"korean japanese mixed", "안녕こんにちは", "annyeongU3053U3093U306BU3061U306F"},
		{"korean chinese mixed", "안녕你好", "annyeongU4F60U597D"},
		{"all languages mixed", "Hello안녕こんにちは你好123", "HelloannyeongU3053U3093U306BU3061U306FU4F60U597D123"},
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
