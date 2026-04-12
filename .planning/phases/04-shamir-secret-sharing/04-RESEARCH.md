# Phase 4: Shamir Secret Sharing - Research

**Researched:** 2026-04-12
**Domain:** Cryptographic library -- Shamir Secret Sharing over GF(256)
**Confidence:** HIGH

## Summary

Phase 4 implements a pure cryptographic library: Shamir Secret Sharing with 2-of-3 threshold scheme operating in GF(256). This is a self-contained, stateless package with zero external dependencies beyond Go's standard `crypto/rand`. No service integration, no gRPC, no storage -- purely mathematical.

The implementation consists of two layers: (1) GF(256) finite field arithmetic using log/exp lookup tables generated at init-time from the AES/Rijndael irreducible polynomial 0x11B, and (2) Split/Combine functions that use polynomial evaluation and Lagrange interpolation over that field. The package location is `twofa/internal/crypto/shamir/` per user decision D-01.

**Primary recommendation:** Implement GF(256) arithmetic first (with property tests), then build Split/Combine on top. Use table-driven tests extensively -- this is a  project where comprehensive testing demonstrates correctness.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Package at `twofa/internal/crypto/shamir/` -- NOT inside `twofaService/`. Crypto is a separate concern from business logic.
- **D-02:** Three files: `gf256.go` (field arithmetic), `shamir.go` (Split/Combine), `shamir_test.go` (tests). GF(256) tests can be in `gf256_test.go` if needed.
- **D-03:** Package-level functions, NOT methods on a struct. Stateless API: `Split(secret []byte, n, threshold int) ([]Share, error)` and `Combine(shares []Share) ([]byte, error)`
- **D-04:** `Share` struct: `Index byte` (x-coordinate: 1, 2, 3) + `Data []byte` (y-values, same length as secret)
- **D-05:** Share indices are 1-based (1, 2, 3). x=0 is reserved for the secret value f(0).
- **D-06:** Input validation: Split returns error for empty secret, n < threshold, threshold < 2, n > 255. Combine returns error for insufficient shares, duplicate indices, empty shares.
- **D-07:** Generator polynomial: 0x11B (x^8 + x^4 + x^3 + x + 1) -- AES/Rijndael polynomial
- **D-08:** Log/exp tables generated at runtime in `init()` function
- **D-09:** Operations: `gfAdd(a, b byte) byte` (XOR), `gfMul(a, b byte) byte` (via log/exp), `gfDiv(a, b byte) byte`, `gfInv(a byte) byte`
- **D-10:** Polynomial evaluation and Lagrange interpolation as internal helpers
- **D-11:** Maximum coverage (~25+ tests) with comprehensive test categories

### Claude's Discretion
- Exact internal helper function decomposition (polynomial evaluation, Lagrange basis)
- Error type design (sentinel errors vs custom types)
- Whether to use `crypto/rand` directly or accept an `io.Reader` for testability
- GF(256) function export level (export arithmetic for testing or keep internal)

### Deferred Ideas (OUT OF SCOPE)
- TOTP integration -- Phase 5
- Secret zeroization -- Phase 7
- Share encryption (AES-256-GCM) -- Phase 6
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CRYPTO-01 | Shamir Secret Sharing implemented from scratch -- Split(secret, n=3, threshold=2) and Combine(shares) in GF(256) | GF(256) arithmetic patterns, polynomial evaluation, Lagrange interpolation code examples |
| CRYPTO-02 | GF(256) arithmetic -- addition via XOR, multiplication via log/exp tables, polynomial evaluation | Log/exp table generation algorithm, init() pattern, arithmetic function signatures |
| CRYPTO-03 | Shamir unit tests -- split->combine roundtrip, any 2-of-3 recovers, 1-of-3 does NOT recover | Test strategy patterns, table-driven test structure, property tests for field axioms |
</phase_requirements>

## Standard Stack

### Core

No external dependencies. This phase uses only Go standard library.

| Package | Source | Purpose | Why Standard |
|---------|--------|---------|--------------|
| `crypto/rand` | stdlib | Random coefficient generation for polynomials | Cryptographically secure randomness required for share security |
| `testing` | stdlib | Unit tests | Standard Go test framework, sufficient for pure math |
| `fmt` | stdlib | Error formatting | Sentinel errors and wrapping |

### Supporting

| Package | Source | Purpose | When to Use |
|---------|--------|---------|-------------|
| `io` | stdlib | `io.Reader` interface | If accepting rand source parameter for testability |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Standard `testing` | `gotest.tools/v3/assert` | Auth service uses it, but standard `testing` is cleaner for math-only tests with no mocks |
| `crypto/rand` directly | `io.Reader` parameter | Accepting `io.Reader` enables deterministic testing; recommend this approach |

**Installation:** None required -- all standard library.

## Architecture Patterns

### Recommended Project Structure

```
twofa/internal/crypto/shamir/
    gf256.go            # GF(256) finite field: log/exp tables, add, mul, div, inv
    gf256_test.go       # Property tests for field axioms
    shamir.go           # Share type, Split, Combine, polynomial helpers
    shamir_test.go      # Roundtrip tests, edge cases, error cases
```

[VERIFIED: twofa/internal/ directory exists, crypto/ does not yet exist]

### Pattern 1: GF(256) Log/Exp Table Generation

**What:** Pre-compute lookup tables for multiplication and division in GF(256) using generator element 3 and irreducible polynomial 0x11B.
**When to use:** At package init time -- tables are constant once computed.

```go
// [ASSUMED] -- standard GF(256) construction algorithm, widely documented
var logTable [256]byte
var expTable [256]byte

func init() {
    x := byte(1)
    for i := 0; i < 255; i++ {
        expTable[i] = x
        logTable[x] = byte(i)
        // Multiply by generator (3) in GF(256)
        x = gfMulNoTable(x, 3)
    }
    expTable[255] = expTable[0] // wrap: exp[255] = 1
    // logTable[0] is undefined (log of 0 doesn't exist)
    // logTable[1] = 0 (already set since exp[0] = 1)
}

// Raw multiplication without tables (used only during init)
func gfMulNoTable(a, b byte) byte {
    var result byte
    for b > 0 {
        if b&1 != 0 {
            result ^= a
        }
        carry := a & 0x80
        a <<= 1
        if carry != 0 {
            a ^= 0x1B // Lower 8 bits of 0x11B
        }
        b >>= 1
    }
    return result
}
```

**Key detail:** The generator element is 3 (commonly used for GF(256) with polynomial 0x11B). The `gfMulNoTable` uses shift-and-XOR (Russian peasant multiplication) with reduction by 0x1B (the lower byte of 0x11B, since bit 8 is implicit in the carry). [ASSUMED -- standard algorithm from AES specification]

### Pattern 2: Polynomial Evaluation (Horner's Method in GF(256))

**What:** Evaluate polynomial f(x) = coeffs[0] + coeffs[1]*x + coeffs[2]*x^2 + ... at a given point.
**When to use:** During Split -- evaluate the random polynomial at each share index.

```go
// [ASSUMED] -- Horner's method adapted for GF(256)
func evalPolynomial(coeffs []byte, x byte) byte {
    // Horner's method: evaluate from highest degree down
    result := byte(0)
    for i := len(coeffs) - 1; i >= 0; i-- {
        result = gfAdd(gfMul(result, x), coeffs[i])
    }
    return result
}
```

### Pattern 3: Lagrange Interpolation at x=0

**What:** Recover f(0) from k points using Lagrange basis polynomials evaluated at x=0.
**When to use:** During Combine -- reconstruct the secret byte from share values.

```go
// [ASSUMED] -- standard Lagrange interpolation, simplified for evaluation at x=0
func lagrangeInterpolateAtZero(xs []byte, ys []byte) byte {
    secret := byte(0)
    k := len(xs)
    for i := 0; i < k; i++ {
        // Compute Lagrange basis polynomial L_i(0)
        // L_i(0) = product of (0 - x_j) / (x_i - x_j) for j != i
        // Since 0 - x_j = x_j in GF(256) (additive inverse = self because XOR)
        basis := byte(1)
        for j := 0; j < k; j++ {
            if i == j {
                continue
            }
            // numerator: x_j (since 0 XOR x_j = x_j)
            // denominator: x_i XOR x_j
            basis = gfMul(basis, gfDiv(xs[j], gfAdd(xs[i], xs[j])))
        }
        secret = gfAdd(secret, gfMul(ys[i], basis))
    }
    return secret
}
```

**Critical GF(256) property:** In GF(2^n), addition and subtraction are the same operation (XOR). So `0 - x_j = x_j` and `x_i - x_j = x_i XOR x_j`. This simplifies the Lagrange formula. [ASSUMED -- fundamental property of characteristic-2 fields]

### Pattern 4: Split Function Structure

**What:** Split a secret byte slice into n shares with given threshold.
**When to use:** The public API entry point.

```go
// [ASSUMED] -- standard Shamir SSS construction
func Split(secret []byte, n, threshold int) ([]Share, error) {
    if len(secret) == 0 {
        return nil, ErrEmptySecret
    }
    if threshold < 2 {
        return nil, ErrThresholdTooLow
    }
    if n < threshold {
        return nil, ErrNotEnoughShares
    }
    if n > 255 {
        return nil, ErrTooManyShares
    }

    shares := make([]Share, n)
    for i := 0; i < n; i++ {
        shares[i] = Share{
            Index: byte(i + 1), // 1-based indices
            Data:  make([]byte, len(secret)),
        }
    }

    // Process each byte of the secret independently
    for byteIdx, secretByte := range secret {
        // Build polynomial: coeffs[0] = secretByte, coeffs[1..threshold-1] = random
        coeffs := make([]byte, threshold)
        coeffs[0] = secretByte

        // Random coefficients for degree 1..threshold-1
        randomBytes := make([]byte, threshold-1)
        if _, err := rand.Read(randomBytes); err != nil {
            return nil, fmt.Errorf("generating random coefficients: %w", err)
        }
        copy(coeffs[1:], randomBytes)

        // Evaluate polynomial at each share index
        for i := 0; i < n; i++ {
            shares[i].Data[byteIdx] = evalPolynomial(coeffs, shares[i].Index)
        }
    }

    return shares, nil
}
```

### Anti-Patterns to Avoid

- **Using int instead of byte for GF(256) elements:** All field elements are 0-255. Using `int` introduces potential overflow/underflow bugs and masks the mathematical constraint. Use `byte` everywhere for field elements. [ASSUMED]
- **Forgetting log(0) is undefined:** `gfMul(0, x)` and `gfMul(x, 0)` must be handled as special cases returning 0 before accessing logTable. Similarly `gfDiv(0, x)` returns 0, and `gfDiv(x, 0)` must return error or panic. [ASSUMED]
- **Using modular arithmetic instead of GF(256):** Regular modular arithmetic (mod 257, mod prime) is NOT the same as GF(256). The project specifies GF(256) -- a Galois field with XOR-based addition and polynomial-based multiplication. [ASSUMED]
- **Reusing polynomial coefficients across bytes:** Each byte of the secret gets its own random polynomial. Reusing coefficients across bytes would leak information. [ASSUMED]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Random byte generation | Custom PRNG | `crypto/rand` | Must be cryptographically secure for share security |
| GF(256) from prime field | Modular arithmetic mod p | Proper GF(2^8) with XOR add and polynomial mul | Characteristic-2 field has different properties |

**Key insight:** This is one of the rare cases where hand-rolling IS the requirement (academic project). But use `crypto/rand` for randomness -- never hand-roll a PRNG.

## Common Pitfalls

### Pitfall 1: Log/Exp Table Off-by-One

**What goes wrong:** The exp table wraps at index 255 (since the multiplicative group has order 255, not 256). Incorrect wrap logic causes multiplication errors.
**Why it happens:** `exp[255]` should equal `exp[0]` (both = 1). The log of 0 is undefined. Off-by-one in modular reduction of log sums.
**How to avoid:** In `gfMul`, compute `(logTable[a] + logTable[b]) % 255` (mod 255, NOT mod 256). In `gfDiv`, compute `(logTable[a] - logTable[b] + 255) % 255` to avoid negative values.
**Warning signs:** `gfMul(a, 1) != a` or `gfMul(a, gfInv(a)) != 1` for some values.
[ASSUMED -- classic implementation bug in GF(256)]

### Pitfall 2: Zero Element Special Cases

**What goes wrong:** Accessing `logTable[0]` returns garbage (log of 0 is undefined in any field). Multiplying or dividing by zero without guard checks causes incorrect results silently.
**Why it happens:** The log table has an entry at index 0, but it's meaningless.
**How to avoid:** Every function that uses logTable must check for zero first: `gfMul` returns 0 if either input is 0; `gfDiv` returns 0 if numerator is 0, returns error if denominator is 0; `gfInv(0)` is an error.
**Warning signs:** Non-zero results from `gfMul(0, x)`.
[ASSUMED]

### Pitfall 3: Lagrange Denominator Zero

**What goes wrong:** During Combine, if two shares have the same index, the denominator `x_i XOR x_j` becomes zero, causing division by zero.
**Why it happens:** Duplicate shares passed to Combine.
**How to avoid:** Validate share indices for uniqueness before interpolation (required by D-06).
**Warning signs:** Panic or incorrect reconstruction with duplicate indices.
[ASSUMED]

### Pitfall 4: Insufficient Shares Appear to Work

**What goes wrong:** Combining 1 share through Lagrange interpolation produces output that looks valid (no error), but the output is wrong. Developer thinks it "partially works."
**Why it happens:** Lagrange interpolation with 1 point for a degree-1 polynomial returns that point's y-value, not the secret. No runtime error occurs.
**How to avoid:** The Combine function should enforce `len(shares) >= threshold` -- but since the package doesn't store the threshold, the minimum is 2 (from D-06: threshold >= 2). Test that 1-share combine either errors or returns wrong data.
**Warning signs:** Test passes with 1 share returning the secret by accident (possible if the share index is carefully chosen -- though in GF(256) this only happens for specific polynomial structures).
[ASSUMED]

### Pitfall 5: Modular Reduction Using 256 Instead of 255

**What goes wrong:** The multiplicative group of GF(256) has order 255 (elements 1-255). Using `% 256` instead of `% 255` in log/exp arithmetic produces incorrect multiplication results for some inputs.
**Why it happens:** Confusion between field size (256 elements) and group order (255 non-zero elements).
**How to avoid:** All log-based arithmetic uses modulo 255: `expTable[(int(logTable[a]) + int(logTable[b])) % 255]`.
**Warning signs:** `gfMul` fails for specific input pairs, especially when log sum equals 255.
[ASSUMED]

## Code Examples

### Complete GF(256) Arithmetic Module

```go
// Source: [ASSUMED] -- standard GF(256) implementation pattern
package shamir

// gfAdd performs addition in GF(256) -- XOR
func gfAdd(a, b byte) byte {
    return a ^ b
}

// gfMul performs multiplication in GF(256) using log/exp tables
func gfMul(a, b byte) byte {
    if a == 0 || b == 0 {
        return 0
    }
    logSum := int(logTable[a]) + int(logTable[b])
    return expTable[logSum%255]
}

// gfDiv performs division in GF(256): a / b
func gfDiv(a, b byte) byte {
    if b == 0 {
        panic("shamir: division by zero in GF(256)")
    }
    if a == 0 {
        return 0
    }
    logDiff := int(logTable[a]) - int(logTable[b]) + 255
    return expTable[logDiff%255]
}

// gfInv returns the multiplicative inverse of a in GF(256)
func gfInv(a byte) byte {
    if a == 0 {
        panic("shamir: inverse of zero in GF(256)")
    }
    return expTable[255-int(logTable[a])]
}
```

### Test Pattern: GF(256) Property Tests

```go
// Source: [ASSUMED] -- standard property-based testing for field axioms
func TestGF256_MulCommutative(t *testing.T) {
    for a := 0; a < 256; a++ {
        for b := 0; b < 256; b++ {
            if gfMul(byte(a), byte(b)) != gfMul(byte(b), byte(a)) {
                t.Fatalf("commutativity failed: %d * %d", a, b)
            }
        }
    }
}

func TestGF256_MulIdentity(t *testing.T) {
    for a := 0; a < 256; a++ {
        if gfMul(byte(a), 1) != byte(a) {
            t.Fatalf("identity failed for %d", a)
        }
    }
}

func TestGF256_MulInverse(t *testing.T) {
    for a := 1; a < 256; a++ { // skip 0
        if gfMul(byte(a), gfInv(byte(a))) != 1 {
            t.Fatalf("inverse failed for %d", a)
        }
    }
}

func TestGF256_Distributive(t *testing.T) {
    for a := 0; a < 256; a++ {
        for b := 0; b < 256; b++ {
            for c := 0; c < 256; c++ {
                // a * (b + c) == a*b + a*c
                lhs := gfMul(byte(a), gfAdd(byte(b), byte(c)))
                rhs := gfAdd(gfMul(byte(a), byte(b)), gfMul(byte(a), byte(c)))
                if lhs != rhs {
                    t.Fatalf("distributive failed: %d * (%d + %d)", a, b, c)
                }
            }
        }
    }
}
```

Note: The exhaustive property tests (256x256x256 for distributive) test all 16M+ combinations. This is feasible since GF(256) is small. For  this is powerful evidence of correctness.

### Test Pattern: Shamir Roundtrip

```go
// Source: [ASSUMED] -- standard Shamir test approach
func TestSplit_Combine_AllPairs(t *testing.T) {
    secret := []byte("test-totp-secret-20b")

    shares, err := Split(secret, 3, 2)
    if err != nil {
        t.Fatalf("Split: %v", err)
    }

    // All 3 combinations of 2-of-3
    pairs := [][2]int{{0, 1}, {0, 2}, {1, 2}}
    for _, pair := range pairs {
        subset := []Share{shares[pair[0]], shares[pair[1]]}
        recovered, err := Combine(subset)
        if err != nil {
            t.Fatalf("Combine(%d,%d): %v", pair[0]+1, pair[1]+1, err)
        }
        if !bytes.Equal(recovered, secret) {
            t.Fatalf("Combine(%d,%d) = %x, want %x", pair[0]+1, pair[1]+1, recovered, secret)
        }
    }
}

func TestCombine_SingleShare_DoesNotRecover(t *testing.T) {
    secret := []byte("secret-data")

    shares, err := Split(secret, 3, 2)
    if err != nil {
        t.Fatalf("Split: %v", err)
    }

    for i, share := range shares {
        recovered, err := Combine([]Share{share})
        // Either error (if we enforce minimum 2) or wrong result
        if err == nil && bytes.Equal(recovered, secret) {
            t.Fatalf("single share %d recovered secret", i+1)
        }
    }
}
```

### Discretion Recommendations

**Error type design:** Use sentinel errors with `var`. Clean, idiomatic Go, easy to test against. [ASSUMED -- Go best practice]

```go
var (
    ErrEmptySecret     = errors.New("shamir: secret must not be empty")
    ErrThresholdTooLow = errors.New("shamir: threshold must be at least 2")
    ErrNotEnoughShares = errors.New("shamir: n must be >= threshold")
    ErrTooManyShares   = errors.New("shamir: n must be <= 255")
    ErrDuplicateIndex  = errors.New("shamir: duplicate share index")
    ErrTooFewShares    = errors.New("shamir: need at least 2 shares to combine")
)
```

**Randomness source:** Accept `io.Reader` parameter for testability, default to `crypto/rand`. This allows deterministic tests. [ASSUMED -- common Go crypto pattern]

```go
// Internal: used by Split
var randReader io.Reader = rand.Reader

// For testing only:
// shamir.randReader = bytes.NewReader(fixedBytes)
```

Alternatively, keep it simple -- use `crypto/rand` directly, test roundtrip behavior rather than deterministic output. Since Split is inherently non-deterministic (random coefficients), roundtrip tests are the natural verification. Recommend this simpler approach.

**GF(256) export level:** Keep arithmetic functions unexported (lowercase). Export only `Split`, `Combine`, `Share`, and error sentinels. Test GF(256) in `gf256_test.go` within the same package (has access to unexported functions). [ASSUMED -- Go package-level test access pattern]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hardcoded log/exp tables | Runtime generation in init() | N/A (decision D-08) | Better for  demonstration |
| Prime-field Shamir (mod p) | GF(256) Shamir | N/A (decision D-07) | Byte-oriented, no big-int arithmetic needed |

**Note:** Shamir Secret Sharing is a stable 1979 algorithm. No "state of the art" changes. The GF(256) variant is widely used (e.g., in SLIP-0039 for BIP-39 seed splitting). The mathematics are timeless. [ASSUMED]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Generator element 3 with polynomial 0x11B produces a valid GF(256) | Architecture Patterns | Table generation produces incorrect field; all arithmetic breaks |
| A2 | `gfMulNoTable` shift-and-XOR algorithm with 0x1B reduction is correct | Architecture Patterns | Init tables are wrong, cascading failures |
| A3 | Lagrange interpolation formula with `0 - x_j = x_j` in GF(2^8) | Architecture Patterns | Combine returns garbage |
| A4 | Multiplicative group order is 255 (mod 255 not mod 256) | Common Pitfalls | Silent multiplication bugs |
| A5 | Horner's method works identically in GF(256) as in regular fields | Architecture Patterns | Polynomial evaluation errors |

**Mitigation for all:** The comprehensive GF(256) property tests (D-11) will catch any arithmetic errors. If `a * inv(a) = 1` for all 255 non-zero elements, and distributivity holds for all 16M triples, the field is correct. Roundtrip Split/Combine tests then validate the higher-level algorithm.

## Open Questions

1. **Combine threshold enforcement**
   - What we know: Combine receives `[]Share` but has no stored threshold
   - What's unclear: Should Combine enforce minimum 2 shares (hardcoded) or just require `len >= 2`?
   - Recommendation: Enforce `len(shares) >= 2` since this package is designed for 2-of-3. If generalized later, add threshold parameter.

2. **Panic vs error for division by zero**
   - What we know: `gfDiv(x, 0)` is mathematically undefined
   - What's unclear: Should internal functions panic (programming error) or return error?
   - Recommendation: Panic for internal functions (gfDiv, gfInv) since zero denominators indicate bugs in the algorithm, not user input errors. Public API (Split, Combine) returns errors for invalid user input.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None needed (Go convention) |
| Quick run command | `cd /Users/vbncursed/programming/2fa/twofa && go test ./internal/crypto/shamir/ -v -count=1` |
| Full suite command | `cd /Users/vbncursed/programming/2fa/twofa && go test ./internal/crypto/shamir/ -v -count=1 -race` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CRYPTO-01 | Split produces n shares, Combine recovers secret | unit | `go test ./internal/crypto/shamir/ -run TestSplit -v` | No -- Wave 0 |
| CRYPTO-02 | GF(256) add/mul/div/inv with log/exp tables | unit | `go test ./internal/crypto/shamir/ -run TestGF256 -v` | No -- Wave 0 |
| CRYPTO-03 | Roundtrip all 2-of-3 combos, 1-of-3 fails | unit | `go test ./internal/crypto/shamir/ -run TestCombine -v` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `cd /Users/vbncursed/programming/2fa/twofa && go test ./internal/crypto/shamir/ -v -count=1`
- **Per wave merge:** `cd /Users/vbncursed/programming/2fa/twofa && go test ./internal/crypto/shamir/ -v -count=1 -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `twofa/internal/crypto/shamir/gf256_test.go` -- covers CRYPTO-02 (field axiom property tests)
- [ ] `twofa/internal/crypto/shamir/shamir_test.go` -- covers CRYPTO-01, CRYPTO-03 (split/combine roundtrip, edge cases)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A (pure crypto library) |
| V3 Session Management | no | N/A |
| V4 Access Control | no | N/A |
| V5 Input Validation | yes | Parameter validation in Split/Combine (D-06) |
| V6 Cryptography | yes | `crypto/rand` for randomness, no custom PRNG |

### Known Threat Patterns for Shamir SSS

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Weak randomness in polynomial coefficients | Information Disclosure | Use `crypto/rand` exclusively, never `math/rand` |
| Side-channel timing on field operations | Information Disclosure | Log/exp table lookups are constant-time by nature; no branching on secret data in hot path |
| Coefficient reuse across Split calls | Information Disclosure | Fresh `crypto/rand` bytes per Split invocation |

## Sources

### Primary (HIGH confidence)
- `workspace/03 - Security/Shamir Secret Sharing.md` -- Project algorithm spec
- `CLAUDE.md` -- Project constraints and architecture rules
- `.planning/phases/04-shamir-secret-sharing/04-CONTEXT.md` -- User decisions
- Existing codebase: `twofa/go.mod`, `twofa/internal/` directory structure

### Secondary (MEDIUM confidence)
- None

### Tertiary (LOW confidence)
- GF(256) implementation details (log/exp table generation, arithmetic) -- all tagged [ASSUMED], verified by property tests at runtime

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- pure stdlib, no external deps, verified go.mod exists
- Architecture: HIGH -- all decisions locked in CONTEXT.md, file structure clear
- Pitfalls: HIGH -- GF(256) pitfalls are well-documented in cryptographic literature (training data), and all are verifiable through exhaustive property tests
- Code examples: MEDIUM -- algorithms are [ASSUMED] from training data, but the test suite (CRYPTO-03) will validate correctness

**Research date:** 2026-04-12
**Valid until:** indefinite -- Shamir SSS (1979) and GF(256) are mathematically stable, no version drift
