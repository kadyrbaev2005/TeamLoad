package domain

import (
	"fmt"
	"strings"
	"unicode"
)

// Initials returns up to two upper-case initials ("Aruzhan Sadykova" → "AS").
func Initials(name string) string {
	var out []rune
	for _, w := range strings.Fields(name) {
		for _, r := range w {
			if unicode.IsLetter(r) {
				out = append(out, unicode.ToUpper(r))
				break
			}
		}
		if len(out) == 2 {
			break
		}
	}
	return string(out)
}

// FirstName returns the first word of a full name.
func FirstName(name string) string {
	if f := strings.Fields(name); len(f) > 0 {
		return f[0]
	}
	return name
}

// SeatColor returns the role colour token for a 1-based seat position.
func SeatColor(position int) string {
	return fmt.Sprintf("--r%d", (position-1)%6+1)
}
