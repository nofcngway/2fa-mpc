# Phase 2: Auth Registration - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-12
**Phase:** 02-auth-registration
**Areas discussed:** Password validation details, Registration response, Error handling strategy, Test coverage scope

---

## Password Validation Details

| Option | Description | Selected |
|--------|-------------|----------|
| ASCII + keyboard rows | Detect ASCII sequences AND keyboard row sequences (qwerty, asdf, zxcv) | ✓ |
| ASCII only | Detect only ASCII-ordered sequences (abcd, 1234) | |
| You decide | Claude chooses | |

**User's choice:** ASCII + keyboard rows
**Notes:** Most thorough approach, covers both sequential ASCII and keyboard row patterns

---

| Option | Description | Selected |
|--------|-------------|----------|
| All rules at once | Collect all violations and return together | ✓ |
| First failure only | Return immediately on first violation | |
| You decide | Claude chooses | |

**User's choice:** All rules at once
**Notes:** Better UX for the client — sees all problems at once

---

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated file in service | password_validation.go in internal/services/authService/ | ✓ |
| Standalone package | Separate pkg/validation/ package | |
| You decide | Claude chooses | |

**User's choice:** Dedicated file in service
**Notes:** Matches CLAUDE.md convention of one file per concern

---

| Option | Description | Selected |
|--------|-------------|----------|
| QWERTY only | Standard QWERTY rows and reverses | |
| QWERTY + numpad | QWERTY rows plus numeric keypad patterns | ✓ |
| You decide | Claude chooses | |

**User's choice:** QWERTY + numpad
**Notes:** More thorough detection including numpad patterns

---

| Option | Description | Selected |
|--------|-------------|----------|
| Case insensitive | Lowercase everything before checking sequences | ✓ |
| Case sensitive | Only exact-case sequences detected | |
| You decide | Claude chooses | |

**User's choice:** Case insensitive
**Notes:** More secure — catches mixed-case sequences like QwErTy

---

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, reject repeats | Reject 4+ identical consecutive characters | ✓ |
| No, only sequences | Only sequential patterns, repeated chars allowed | |
| You decide | Claude decides | |

**User's choice:** Yes, reject repeats
**Notes:** Additional security measure for patterns like aaaa, 1111

---

## Registration Response

| Option | Description | Selected |
|--------|-------------|----------|
| User ID only | Return just user_id and created_at | |
| User ID + tokens | Return user_id AND JWT tokens immediately | ✓ (intent) |
| You decide | Claude chooses | |

**User's choice:** User ID + tokens (but phased)
**Notes:** User wants tokens ultimately, but agreed to return user_id/email/created_at in Phase 2, add tokens in Phase 3

---

| Option | Description | Selected |
|--------|-------------|----------|
| Return user_id now, add tokens in Phase 3 | Clean phasing | ✓ |
| Stub tokens now | Placeholder token fields | |
| You decide | Claude decides | |

**User's choice:** Return user_id now, add tokens in Phase 3

---

| Option | Description | Selected |
|--------|-------------|----------|
| Basic format check | Simple regex: contains @, has domain, no spaces | ✓ |
| RFC 5322 compliant | Full RFC-compliant email validation | |
| You decide | Claude picks | |

**User's choice:** Basic format check

---

## Error Handling Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Per-rule messages | Return specific messages for each failed rule | ✓ |
| Generic message | Single 'password does not meet policy' | |
| Rules list in details | Generic status + structured metadata | |

**User's choice:** Per-rule messages
**Notes:** Helps client display actionable feedback

---

| Option | Description | Selected |
|--------|-------------|----------|
| Reveal duplicate | Return 'user with this email already exists' | ✓ |
| Generic error | Return generic 'registration failed' | |
| You decide | Claude chooses | |

**User's choice:** Reveal duplicate
**Notes:** Acceptable tradeoff for academic project

---

| Option | Description | Selected |
|--------|-------------|----------|
| Concatenated string | Join all failing rules into one message | |
| Separate error codes | Define specific error types per rule | ✓ |
| You decide | Claude picks | |

**User's choice:** Separate error codes
**Notes:** More extensible — ErrPasswordTooShort, ErrMissingUppercase, etc.

---

## Test Coverage Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Table-driven tests | Go idiomatic test cases table | ✓ |
| Separate test functions | One TestXxx per rule | |
| You decide | Claude picks | |

**User's choice:** Table-driven tests

---

| Option | Description | Selected |
|--------|-------------|----------|
| Both layers | Password validation AND service-level register tests | ✓ |
| Password validation only | Focus on password_validation.go only | |
| You decide | Claude decides | |

**User's choice:** Both layers

---

| Option | Description | Selected |
|--------|-------------|----------|
| Comprehensive | 3 vs 4 boundary for EACH type (~12+ cases) | ✓ |
| Representative | 3 vs 4 for ASCII and keyboard only (~6-8 cases) | |
| You decide | Claude decides | |

**User's choice:** Comprehensive
**Notes:** Full boundary coverage for all sequence types
