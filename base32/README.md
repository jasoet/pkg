# Base32 Package

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v3/base32.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v3/base32)

Crockford Base32 encoding and CRC-10 checksums for human-readable, error-correcting identifiers.

## Features

- **Crockford Base32 Encoding**
  - Human-readable alphabet (excludes ambiguous characters: I, L, O, U)
  - Case-insensitive decoding
  - Automatic error correction (I→1, L→1, O→0)
  - Fixed-length and compact encoding modes
  - URL-safe output

- **CRC-10 Checksums**
  - 99.9%+ error detection rate
  - Detects single character errors (100%)
  - Detects transpositions (99.9%+)
  - Detects double errors (99.9%+)
  - Only 2 characters overhead

## Installation

```bash
go get github.com/jasoet/pkg/v3
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/jasoet/pkg/v3/base32"
)

func main() {
    // Encode a number
    id, err := base32.EncodeBase32(12345, 8)  // "00000C1S", nil
    if err != nil {
        panic(err)
    }

    // Add checksum for error detection
    idWithChecksum, err := base32.AppendChecksum(id)
    if err != nil {
        panic(err)
    }

    // Validate checksum
    if base32.ValidateChecksum(idWithChecksum) {
        fmt.Println("Valid ID!")
    }

    // Decode back
    value, _ := base32.DecodeBase32(id)
    fmt.Println(value)  // 12345
}
```

## Use Cases

### 1. URL Shorteners

```go
// Database ID to short code
databaseID := uint64(123456789)
shortCode := base32.EncodeBase32Compact(databaseID)
// https://short.url/3NQK8N

// Decode back
decoded, _ := base32.DecodeBase32(shortCode)  // 123456789
```

### 2. Order/Transaction IDs

```go
// Generate order ID with timestamp and sequence
timestamp := uint64(time.Now().Unix())
sequence := uint64(12345)

timeCode, _ := base32.EncodeBase32(timestamp, 8)
seqCode, _ := base32.EncodeBase32(sequence, 4)  // "0C1S"

// AppendChecksum normalizes its input first: dashes are removed and
// common lookalikes are corrected (note: "ORD" contains O, which → 0)
orderID, _ := base32.AppendChecksum("ORD-" + timeCode + "-" + seqCode)
// "0RD" + timeCode + seqCode + 2-char checksum, e.g. "0RD01N62VHA0C1S27"
```

### 3. License Keys

```go
productID := uint64(42)
customerID := uint64(789)

product, _ := base32.EncodeBase32(productID, 2)    // "1A"
customer, _ := base32.EncodeBase32(customerID, 4)  // "00RN"

licenseKey, _ := base32.AppendChecksum(product + customer)
// "1A00RN7D"
```

### 4. Voucher/Coupon Codes

```go
voucherID := uint64(9999)
code, _ := base32.EncodeBase32(voucherID, 4)           // "09RF"
codeWithChecksum, _ := base32.AppendChecksum(code)     // "09RFZB"
// Easy to type, error-correcting
```

### 5. IoT Device IDs

```go
deviceSerial := uint64(123456)
deviceID := base32.EncodeBase32Compact(deviceSerial)
// Compact, human-readable device identifier
```

## API Reference

### Base32 Encoding

#### `EncodeBase32(value uint64, length int) (string, error)`

Encodes an unsigned integer to a fixed-length Base32 string.

```go
encoded, err := base32.EncodeBase32(42, 4)     // "001A", nil
encoded, err := base32.EncodeBase32(12345, 6)  // "000C1S", nil
```

#### `EncodeBase32Compact(value uint64) string`

Encodes to the minimum number of characters needed (no error return).

```go
encoded := base32.EncodeBase32Compact(0)      // "0"
encoded := base32.EncodeBase32Compact(12345)  // "C1S"
```

#### `DecodeBase32(encoded string) (uint64, error)`

Decodes a Base32 string to an unsigned integer.

```go
value, err := base32.DecodeBase32("C1S")  // 12345, nil
value, err := base32.DecodeBase32("c1s")  // 12345, nil (case-insensitive)
value, err := base32.DecodeBase32("I0")   // 32, nil (I→1 correction)
```

#### `NormalizeBase32(input string) string`

Normalizes Base32 input by:
- Converting to uppercase
- Removing dashes and spaces
- Correcting common mistakes (I→1, L→1, O→0)

```go
base32.NormalizeBase32("abc-def")  // "ABCDEF"
base32.NormalizeBase32("1O 2I")    // "1021"
```

#### `IsValidBase32Char(c rune) bool`

Checks if a character is valid in Base32 encoding.

```go
base32.IsValidBase32Char('A')  // true
base32.IsValidBase32Char('O')  // true (auto-corrected)
base32.IsValidBase32Char('U')  // false
```

### Checksums

#### `CalculateChecksum(data string) (string, error)`

Computes a 2-character CRC-10 checksum.

```go
checksum, err := base32.CalculateChecksum("ABC123")  // "TF", nil
```

#### `AppendChecksum(data string) (string, error)`

Adds checksum to the end of data. The input is normalized via
`NormalizeBase32` first (uppercased, dashes/spaces removed, I→1 / L→1 / O→0),
so dashed or lowercase identifiers work; clean input is unaffected.

```go
withChecksum, err := base32.AppendChecksum("ABC123")     // "ABC123TF", nil
withChecksum, err := base32.AppendChecksum("0000-c1p9")  // "0000C1P9Q0", nil
```

#### `ValidateChecksum(input string) bool`

Verifies checksum validity. The input is normalized via `NormalizeBase32`
first, so dashed or lowercase checksummed strings validate; clean input is
unaffected.

```go
valid := base32.ValidateChecksum("ABC123TF")    // true
valid := base32.ValidateChecksum("abc-123-tf")  // true (normalized)
valid := base32.ValidateChecksum("ABC123ZZ")    // false
```

#### `StripChecksum(input string) string`

Removes the last 2 characters (checksum).

```go
data := base32.StripChecksum("ABC123TF")  // "ABC123"
```

#### `ExtractChecksum(input string) string`

Extracts the last 2 characters (checksum).

```go
checksum := base32.ExtractChecksum("ABC123TF")  // "TF"
```

## Error Detection

The CRC-10 (CRC-10/ATM) checksum provides strong error detection:

| Error Type | Detection Rate |
|------------|----------------|
| Single character error | 100% |
| Transposition (AB→BA) | 99.9%+ |
| Double errors | 99.9%+ |
| Insertion/deletion (non-leading-zero) | High |

### Known limitation: leading zeros

Because the CRC register is initialized to zero, inserting or deleting **leading
`0` characters is invisible to the checksum**:

```go
a, _ := base32.CalculateChecksum("C1S")     // same as ...
b, _ := base32.CalculateChecksum("000C1S")  // ... this
// a == b

base32.ValidateChecksum("00")           // true  (all-zero string checksums to "00")
base32.ValidateChecksum("0000000000")   // true
```

Do not rely on the checksum to catch loss or addition of leading zeros. When
that matters, store and compare identifiers at a **fixed length** (see
`EncodeBase32`) so leading zeros are structurally significant. This zero-init
behavior is a compatibility contract pinned by the package's golden vectors and
will not change within the v3 series.

### Example

```go
// Valid ID
validID, _ := base32.AppendChecksum("ABC123")

// Corrupted ID (A → X)
corrupted := "XBC123" + base32.ExtractChecksum(validID)
base32.ValidateChecksum(corrupted)  // false - detected!

// Transposition (AB → BA)
chars := []rune(validID)
chars[0], chars[1] = chars[1], chars[0]
transposed := string(chars)
base32.ValidateChecksum(transposed)  // false - detected!
```

## Examples

Run the comprehensive walkthrough (no build tag needed):

```bash
go run ./examples/base32
```

Or the compact demo in this directory (requires the `example` build tag):

```bash
go run -tags=example ./base32/examples
```

See [examples/main.go](examples/main.go) and [../examples/base32/example.go](../examples/base32/example.go) for detailed usage patterns.

## Performance

Benchmarks on modern hardware:

```
BenchmarkEncodeBase32-8          50000000    25.3 ns/op
BenchmarkDecodeBase32-8          30000000    45.2 ns/op
BenchmarkCalculateChecksum-8     10000000   125.0 ns/op
BenchmarkValidateChecksum-8       8000000   160.0 ns/op
```

## Alphabet Reference

### Crockford Base32 Alphabet

```
0 1 2 3 4 5 6 7 8 9 A B C D E F G H J K M N P Q R S T V W X Y Z
```

**Excluded characters:**
- `I` - Looks like `1` (auto-corrected to `1`)
- `L` - Looks like `1` (auto-corrected to `1`)
- `O` - Looks like `0` (auto-corrected to `0`)
- `U` - Could be confused with `V`

This design minimizes human transcription errors.

## Best Practices

1. **Always use checksums for user-facing IDs**
   ```go
   // ✓ Good
   id, err := base32.AppendChecksum(data)

   // ✗ Avoid
   id := data  // No error detection
   ```

2. **Normalize user input**
   ```go
   userInput := "ab-cd-ef"
   normalized := base32.NormalizeBase32(userInput)
   ```

3. **Use fixed-length encoding for databases**
   ```go
   // Consistent length for indexing
   id := base32.EncodeBase32(value, 10)
   ```

4. **Use compact encoding for URLs**
   ```go
   // Shorter URLs
   shortCode := base32.EncodeBase32Compact(value)
   ```

## Migration from tix-core

If migrating from `github.com/jasoet/tix-core/encoding`:

**Before:**
```go
import "github.com/jasoet/tix-core/encoding"
```

**After:**
```go
import "github.com/jasoet/pkg/v3/base32"
```

API is 100% compatible - only the import path and package name change.

## Contributing

See the main [pkg/v3 repository](https://github.com/jasoet/pkg) for contribution guidelines.

## License

MIT License - see [LICENSE](../LICENSE) for details.

---

**Part of [github.com/jasoet/pkg/v3](https://github.com/jasoet/pkg)** - Production-ready Go utility packages.
