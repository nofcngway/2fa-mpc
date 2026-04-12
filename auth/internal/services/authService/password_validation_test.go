package authService

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantErr   bool
		wantRules []error
	}{
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
			wantRules: []error{ErrPasswordTooShort},
		},
		{
			name:      "too short 9 chars",
			password:  "Short1!aB",
			wantErr:   true,
			wantRules: []error{ErrPasswordTooShort},
		},

		// Missing character classes
		{
			name:      "missing uppercase",
			password:  "alllowercase1!ab",
			wantErr:   true,
			wantRules: []error{ErrMissingUppercase},
		},
		{
			name:      "missing lowercase",
			password:  "ALLUPPERCASE1!AB",
			wantErr:   true,
			wantRules: []error{ErrMissingLowercase},
		},
		{
			name:      "missing digit",
			password:  "NoDigitsHere!!Ab",
			wantErr:   true,
			wantRules: []error{ErrMissingDigit},
		},
		{
			name:      "missing special char",
			password:  "NoSpecial1Charrr",
			wantErr:   true,
			wantRules: []error{ErrMissingSpecialChar},
		},

		// ASCII sequential (4+)
		{
			name:      "sequential abcd ascending",
			password:  "Te$t00abcdXY",
			wantErr:   true,
			wantRules: []error{ErrSequentialChars},
		},
		{
			name:      "sequential dcba descending",
			password:  "Te$t00dcbaXY",
			wantErr:   true,
			wantRules: []error{ErrSequentialChars},
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
			wantRules: []error{ErrSequentialChars},
		},
		{
			name:      "sequential rewq reversed",
			password:  "Te$t00rewqXY",
			wantErr:   true,
			wantRules: []error{ErrSequentialChars},
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
			wantRules: []error{ErrSequentialChars},
		},
		{
			name:      "sequential fdsa row 2 reversed",
			password:  "Te$t00fdsaXY",
			wantErr:   true,
			wantRules: []error{ErrSequentialChars},
		},

		// QWERTY row 3
		{
			name:      "sequential zxcv row 3",
			password:  "Te$t00zxcvXY",
			wantErr:   true,
			wantRules: []error{ErrSequentialChars},
		},

		// Numpad
		{
			name:      "sequential 7894 numpad",
			password:  "Te$tAA7894xY",
			wantErr:   true,
			wantRules: []error{ErrSequentialChars},
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
			wantRules: []error{ErrSequentialChars},
		},
		{
			name:      "sequential 4321 digits descending",
			password:  "Te$tAA4321xY",
			wantErr:   true,
			wantRules: []error{ErrSequentialChars},
		},

		// Case-insensitive sequential
		{
			name:      "sequential ABCD case insensitive",
			password:  "te$t00ABCDxY",
			wantErr:   true,
			wantRules: []error{ErrSequentialChars},
		},

		// Repeated characters
		{
			name:      "4 repeated aaaa",
			password:  "Te$t00aaaaXY",
			wantErr:   true,
			wantRules: []error{ErrRepeatedChars},
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
			wantRules: []error{ErrRepeatedChars},
		},

		// Multi-error case
		{
			name:     "short and missing classes",
			password: "ab",
			wantErr:  true,
			wantRules: []error{
				ErrPasswordTooShort,
				ErrMissingUppercase,
				ErrMissingDigit,
				ErrMissingSpecialChar,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}

				var validationErr *PasswordValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("expected *PasswordValidationError, got %T", err)
				}

				for _, wantRule := range tt.wantRules {
					found := false
					for _, violation := range validationErr.Violations {
						if errors.Is(violation, wantRule) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected violation %v not found in %v", wantRule, validationErr.Violations)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			}
		})
	}
}
