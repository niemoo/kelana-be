package bookingcode

import (
	"strings"
	"testing"
)

func TestAlphabet(t *testing.T) {
	if len(Alphabet) != 32 {
		t.Errorf("len(Alphabet) = %d, want 32", len(Alphabet))
	}
	for _, c := range "0O1I" {
		if strings.ContainsRune(Alphabet, c) {
			t.Errorf("Alphabet contains %q, which BR-10 forbids", c)
		}
	}
	seen := make(map[rune]bool)
	for _, c := range Alphabet {
		if seen[c] {
			t.Errorf("Alphabet contains %q twice", c)
		}
		seen[c] = true
	}
}

func TestGenerateFormat(t *testing.T) {
	for range 1000 {
		code := Generate()
		if len(code) != Length {
			t.Fatalf("Generate() = %q, want %d characters", code, Length)
		}
		for _, c := range code {
			if !strings.ContainsRune(Alphabet, c) {
				t.Fatalf("Generate() = %q contains %q, which is not in Alphabet", code, c)
			}
		}
	}
}

func TestGenerateIsRandom(t *testing.T) {
	seen := make(map[string]bool)
	for range 100 {
		seen[Generate()] = true
	}
	// Ada 32^6 (sekitar 1 miliar) kemungkinan kode, jadi dari 100 kode
	// hampir mustahil ada lebih dari 5 yang kembar.
	if len(seen) < 95 {
		t.Errorf("only %d distinct codes out of 100; Generate() is not random enough", len(seen))
	}
}
