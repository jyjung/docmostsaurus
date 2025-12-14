package main

import (
	"fmt"

	"github.com/jung/doc2git/internal/hangul"
)

func main() {
	words := []string{"하이하이", "안녕", "값", "Google", "123"}

	for _, w := range words {
		fmt.Printf("%s -> %s\n", w, hangul.Romanize(w))
	}
}
