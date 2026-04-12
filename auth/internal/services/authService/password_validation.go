package authService

import (
	"errors"
	"strings"
	"unicode"
)

// ErrPasswordTooShort indicates the password is shorter than 12 characters.
var ErrPasswordTooShort = errors.New("password must be at least 12 characters")

// ErrMissingUppercase indicates the password has no uppercase letter.
var ErrMissingUppercase = errors.New("password must contain at least one uppercase letter")

// ErrMissingLowercase indicates the password has no lowercase letter.
var ErrMissingLowercase = errors.New("password must contain at least one lowercase letter")

// ErrMissingDigit indicates the password has no digit.
var ErrMissingDigit = errors.New("password must contain at least one digit")

// ErrMissingSpecialChar indicates the password has no special character.
var ErrMissingSpecialChar = errors.New("password must contain at least one special character")

// ErrSequentialChars indicates the password contains 4 or more sequential characters.
var ErrSequentialChars = errors.New("password must not contain 4 or more sequential characters")

// ErrRepeatedChars indicates the password contains 4 or more identical consecutive characters.
var ErrRepeatedChars = errors.New("password must not contain 4 or more identical consecutive characters")

// PasswordValidationError aggregates all password validation violations.
type PasswordValidationError struct {
	Violations []error
}

// Error returns a semicolon-separated string of all violation messages.
func (e *PasswordValidationError) Error() string {
	msgs := make([]string, len(e.Violations))
	for i, v := range e.Violations {
		msgs[i] = v.Error()
	}
	return strings.Join(msgs, "; ")
}

// MIN_PASSWORD_LENGTH is the minimum required password length.
const MIN_PASSWORD_LENGTH = 12

// SEQUENCE_THRESHOLD is the minimum number of sequential or repeated characters to trigger rejection.
const SEQUENCE_THRESHOLD = 4

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

	if len(password) < MIN_PASSWORD_LENGTH {
		violations = append(violations, ErrPasswordTooShort)
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
		violations = append(violations, ErrMissingUppercase)
	}
	if !hasLower {
		violations = append(violations, ErrMissingLowercase)
	}
	if !hasDigit {
		violations = append(violations, ErrMissingDigit)
	}
	if !hasSpecial {
		violations = append(violations, ErrMissingSpecialChar)
	}

	if containsSequential(password, SEQUENCE_THRESHOLD) {
		violations = append(violations, ErrSequentialChars)
	}

	if containsRepeated(password, SEQUENCE_THRESHOLD) {
		violations = append(violations, ErrRepeatedChars)
	}

	if len(violations) > 0 {
		return &PasswordValidationError{Violations: violations}
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
	for i := 0; i <= len(password)-winSize; i++ {
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
	for i := 0; i <= len(lower)-minLen; i++ {
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
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
