package utils

import (
	"strings"
	"testing"
	"unicode"
)

func TestRandomString(t *testing.T) {
	t.Run("correct length", func(t *testing.T) {
		for _, n := range []int{0, 1, 8, 16, 64} {
			got := RandomString(n)
			if len(got) != n {
				t.Errorf("RandomString(%d) length = %d, want %d", n, len(got), n)
			}
		}
	})

	t.Run("only letters", func(t *testing.T) {
		s := RandomString(200)
		for _, r := range s {
			if !unicode.IsLetter(r) {
				t.Errorf("RandomString produced non-letter character: %q", r)
			}
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		seen := make(map[string]struct{}, 20)
		for range 20 {
			s := RandomString(16)
			seen[s] = struct{}{}
		}
		// Expect at least 10 unique strings out of 20 (collision is astronomically unlikely)
		if len(seen) < 10 {
			t.Errorf("RandomString produced too many collisions: %d unique out of 20", len(seen))
		}
	})

	t.Run("only ascii letters", func(t *testing.T) {
		const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		s := RandomString(100)
		for _, r := range s {
			if !strings.ContainsRune(alphabet, r) {
				t.Errorf("RandomString produced character outside alphabet: %q", r)
			}
		}
	})
}
