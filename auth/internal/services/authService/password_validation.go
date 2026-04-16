package authService

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

// minPasswordLength is the minimum required password length.
const minPasswordLength = 12

// maxPasswordLength caps input to prevent bcrypt truncation (72 bytes) and CPU DoS.
const maxPasswordLength = 128

// sequenceThreshold is the minimum number of sequential or repeated characters to trigger rejection.
const sequenceThreshold = 4

// sequences defines character sequences to check against (ASCII, QWERTY rows, numpad).
var sequences = []string{
	"abcdefghijklmnopqrstuvwxyz",
	"0123456789",
	"qwertyuiop",
	"asdfghjkl",
	"zxcvbnm",
	"7894561230",
}

// ValidatePassword checks a password against all validation rules and returns
// all violations simultaneously via PasswordValidationError.
func ValidatePassword(password string) error {
	var violations []error

	runeCount := utf8.RuneCountInString(password)
	if runeCount < minPasswordLength {
		violations = append(violations, domain.ErrPasswordTooShort)
	}
	if runeCount > maxPasswordLength {
		violations = append(violations, domain.ErrPasswordTooLong)
	}

	hasUpper, hasLower, hasDigit, hasSpecial := false, false, false, false
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case !unicode.IsLetter(ch) && !unicode.IsDigit(ch):
			hasSpecial = true
		}
	}

	if !hasUpper {
		violations = append(violations, domain.ErrMissingUppercase)
	}
	if !hasLower {
		violations = append(violations, domain.ErrMissingLowercase)
	}
	if !hasDigit {
		violations = append(violations, domain.ErrMissingDigit)
	}
	if !hasSpecial {
		violations = append(violations, domain.ErrMissingSpecialChar)
	}

	if containsSequential(password, sequenceThreshold) {
		violations = append(violations, domain.ErrSequentialChars)
	}

	if containsRepeated(password, sequenceThreshold) {
		violations = append(violations, domain.ErrRepeatedChars)
	}

	if len(violations) > 0 {
		return &domain.PasswordValidationError{Violations: violations}
	}
	return nil
}

// containsSequential checks if the password contains minLen or more consecutive
// characters from any known sequence (forward or reversed), case-insensitively.
func containsSequential(password string, minLen int) bool {
	lower := strings.ToLower(password)

	for _, seq := range sequences {
		if containsSubseq(lower, seq, minLen) {
			return true
		}
		rev := reverse(seq)
		if containsSubseq(lower, rev, minLen) {
			return true
		}
	}
	return false
}

// containsSubseq checks if any sliding window of size winSize in the password
// appears as a substring of the given sequence.
func containsSubseq(password string, seq string, winSize int) bool {
	if len(password) < winSize {
		return false
	}
	for i := range len(password) - winSize + 1 {
		window := password[i : i+winSize]
		if strings.Contains(seq, window) {
			return true
		}
	}
	return false
}

// containsRepeated checks if the password contains minLen or more identical
// consecutive characters (case-insensitive).
func containsRepeated(password string, minLen int) bool {
	lower := strings.ToLower(password)
	if len(lower) < minLen {
		return false
	}
	for i := range len(lower) - minLen + 1 {
		allSame := true
		for j := 1; j < minLen; j++ {
			if lower[i+j] != lower[i] {
				allSame = false
				break
			}
		}
		if allSame {
			return true
		}
	}
	return false
}

// reverse returns the string with its characters in reverse order.
func reverse(s string) string {
	runes := []rune(s)
	slices.Reverse(runes)
	return string(runes)
}
