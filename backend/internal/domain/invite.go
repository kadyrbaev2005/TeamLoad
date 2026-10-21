package domain

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// InviteAlphabet excludes look-alike characters (I, O, 0, 1).
const InviteAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// InviteCodeLength is the number of characters in a code.
const InviteCodeLength = 6

// NewInviteCode returns a random code from InviteAlphabet.
func NewInviteCode() string {
	b := make([]byte, InviteCodeLength)
	max := big.NewInt(int64(len(InviteAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		b[i] = InviteAlphabet[n.Int64()]
	}
	return string(b)
}

// NormalizeInviteCode trims and upper-cases user input.
func NormalizeInviteCode(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

// ValidInviteCode reports whether s has the code format.
func ValidInviteCode(s string) bool {
	if len(s) != InviteCodeLength {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune(InviteAlphabet, c) {
			return false
		}
	}
	return true
}
