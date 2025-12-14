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
