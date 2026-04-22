package auth_service

import (
	"errors"
	"slices"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/vbncursed/vkr/auth/internal/domain"
)

type passwordValidationCase struct {
	name      string
	password  string
	wantErr   bool
	wantRules []error
}

func passwordValidationSuite() []passwordValidationCase {
	return []passwordValidationCase{
		// Valid passwords
		{
			name:     "valid strong password",
			password: "MyStr0ng!Pass99",
			wantErr:  false,
		},
		{
			name:     "valid password exactly 12 chars",
			password: "C0mpl3x#Pass",
			wantErr:  false,
		},
		{
			name:     "valid password alternative",
			password: "C0mpl3x#Passw",
			wantErr:  false,
		},

		// Length validation
		{
			name:      "too short 11 chars",
			password:  "Short1!aaBc",
			wantErr:   true,
			wantRules: []error{domain.ErrPasswordTooShort},
		},
		{
			name:      "too short 9 chars",
			password:  "Short1!aB",
			wantErr:   true,
			wantRules: []error{domain.ErrPasswordTooShort},
		},

		// Missing character classes
		{
			name:      "missing uppercase",
			password:  "alllowercase1!ab",
			wantErr:   true,
			wantRules: []error{domain.ErrMissingUppercase},
		},
		{
			name:      "missing lowercase",
			password:  "ALLUPPERCASE1!AB",
			wantErr:   true,
			wantRules: []error{domain.ErrMissingLowercase},
		},
		{
			name:      "missing digit",
			password:  "NoDigitsHere!!Ab",
			wantErr:   true,
			wantRules: []error{domain.ErrMissingDigit},
		},
		{
			name:      "missing special char",
			password:  "NoSpecial1Charrr",
			wantErr:   true,
			wantRules: []error{domain.ErrMissingSpecialChar},
		},

		// ASCII sequential (4+)
		{
			name:      "sequential abcd ascending",
			password:  "Te$t00abcdXY",
			wantErr:   true,
			wantRules: []error{domain.ErrSequentialChars},
		},
		{
			name:      "sequential dcba descending",
			password:  "Te$t00dcbaXY",
			wantErr:   true,
			wantRules: []error{domain.ErrSequentialChars},
		},
		{
			name:     "3 sequential abc allowed",
			password: "Te$t00abc0XY",
			wantErr:  false,
		},

		// QWERTY row 1
		{
			name:      "sequential qwer forward",
			password:  "Te$t00qwerXY",
			wantErr:   true,
			wantRules: []error{domain.ErrSequentialChars},
		},
		{
			name:      "sequential rewq reversed",
			password:  "Te$t00rewqXY",
			wantErr:   true,
			wantRules: []error{domain.ErrSequentialChars},
		},
		{
			name:     "3 qwerty qwe allowed",
			password: "Te$t00qwe0XY",
			wantErr:  false,
		},

		// QWERTY row 2
		{
			name:      "sequential asdf row 2",
			password:  "Te$t00asdfXY",
			wantErr:   true,
			wantRules: []error{domain.ErrSequentialChars},
		},
		{
			name:      "sequential fdsa row 2 reversed",
			password:  "Te$t00fdsaXY",
			wantErr:   true,
			wantRules: []error{domain.ErrSequentialChars},
		},

		// QWERTY row 3
		{
			name:      "sequential zxcv row 3",
			password:  "Te$t00zxcvXY",
			wantErr:   true,
			wantRules: []error{domain.ErrSequentialChars},
		},

		// Numpad
		{
			name:      "sequential 7894 numpad",
			password:  "Te$tAA7894xY",
			wantErr:   true,
			wantRules: []error{domain.ErrSequentialChars},
		},
		{
			name:     "3 numpad 789 allowed",
			password: "Te$tAA789xxY",
			wantErr:  false,
		},

		// Digit sequences
		{
			name:      "sequential 1234 digits ascending",
			password:  "Te$tAA1234xY",
			wantErr:   true,
			wantRules: []error{domain.ErrSequentialChars},
		},
		{
			name:      "sequential 4321 digits descending",
			password:  "Te$tAA4321xY",
			wantErr:   true,
			wantRules: []error{domain.ErrSequentialChars},
		},

		// Case-insensitive sequential
		{
			name:      "sequential ABCD case insensitive",
			password:  "te$t00ABCDxY",
			wantErr:   true,
			wantRules: []error{domain.ErrSequentialChars},
		},

		// Repeated characters
		{
			name:      "4 repeated aaaa",
			password:  "Te$t00aaaaXY",
			wantErr:   true,
			wantRules: []error{domain.ErrRepeatedChars},
		},
		{
			name:     "3 repeated aaa allowed",
			password: "Te$t00aaa0XY",
			wantErr:  false,
		},
		{
			name:      "4 repeated 1111",
			password:  "Te$tAA1111xY",
			wantErr:   true,
			wantRules: []error{domain.ErrRepeatedChars},
		},

		// Multi-error case
		{
			name:     "short and missing classes",
			password: "ab",
			wantErr:  true,
			wantRules: []error{
				domain.ErrPasswordTooShort,
				domain.ErrMissingUppercase,
				domain.ErrMissingDigit,
				domain.ErrMissingSpecialChar,
			},
		},
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()
	for _, tt := range passwordValidationSuite() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePassword(tt.password)

			if !tt.wantErr {
				assert.NilError(t, err)
				return
			}

			assert.Assert(t, err != nil, "expected error but got nil")

			validationErr, ok := errors.AsType[*domain.PasswordValidationError](err)
			assert.Assert(t, ok, "expected *domain.PasswordValidationError, got %T", err)

			assert.Equal(t, len(validationErr.Violations), len(tt.wantRules),
				"expected %d violations, got %d: %v", len(tt.wantRules), len(validationErr.Violations), validationErr.Violations)

			for _, wantRule := range tt.wantRules {
				found := slices.ContainsFunc(validationErr.Violations, func(v error) bool {
					return errors.Is(v, wantRule)
				})
				assert.Assert(t, found, "expected violation %v not found in %v", wantRule, validationErr.Violations)
			}
		})
	}
}
