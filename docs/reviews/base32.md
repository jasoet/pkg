# Code Review: `base32` Package

**Date:** 2026-03-21

## Package Summary

Implements Crockford Base32 encoding (32-character alphabet `0-9A-Z` excluding I, L, O, U) and a CRC-10 checksum over Base32 strings. Primary purpose: generating human-readable, error-correcting identifiers (order IDs, license keys, URL short codes).

---

## Issues Found

### Medium

**M1 — `CalculateChecksum` silently accepts empty string** (`checksum.go:32`)

Empty `data` iterates zero times and returns `("00", nil)`. But `ValidateChecksum("00")` returns `false` because `len("00") < 3`. Confusing round-trip behavior.

**Fix:** Add `if data == "" { return "", fmt.Errorf("empty Base32 string") }`.

**M2 — `NormalizeBase32` godoc comment misplaced** (`base32.go:175-199`)

Comment is above `normalizeReplacer` var, not above the function. `go doc` shows no documentation for the function.

**Fix:** Move comment to immediately precede `func NormalizeBase32`.

**M3 — README has multiple documentation inaccuracies**

API Reference shows functions returning single `string`, omitting `error` return. Quick Start examples wouldn't compile.

### Low

- L1: Dead code — second overflow check in `DecodeBase32` is mathematically unreachable
- L2: `EncodeBase32Compact` uses O(N^2) prepend pattern (negligible for uint64 — max 13 iterations)
- L3: Test uses non-printable sub-test names (`string(rune(0))`)
- L4: `NormalizeBase32` only strips hyphen and space — tab/newline not handled

### Security

- **No vulnerabilities found.** No timing attacks relevant (CRC-10 is not cryptographic; only 1024 possible values).
- All mutable state is local to function calls. Package-level variables read-only after init.
- No race conditions (confirmed with `-race`).
- Input validation present and correct in all encode/decode paths.

### Recommendations

1. Reject empty string in `CalculateChecksum`
2. Move `NormalizeBase32` doc comment above the function
3. Fix README function signatures and examples
4. Remove unreachable dead code in `DecodeBase32`
5. Consider adding `\t`, `\n`, `\r` to `normalizeReplacer`
