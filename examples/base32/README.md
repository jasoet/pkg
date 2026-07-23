# Base32 Package Examples

This directory contains examples demonstrating how to use the `base32` package for Crockford Base32 encoding and CRC-10 checksums in Go applications.

## 📍 Example Code Location

**Full example implementation:** [example.go](example.go) in this directory.

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern
import "github.com/jasoet/pkg/v3/base32"

// Encoding
encoded, err := base32.EncodeBase32(12345, 8)  // Fixed-length: "00000C1S"
compact := base32.EncodeBase32Compact(12345)   // Compact: "C1S"

// Decoding
value, err := base32.DecodeBase32("C1S")        // Returns: 12345

// Checksums (AppendChecksum/ValidateChecksum normalize their input first)
withChecksum, err := base32.AppendChecksum("ABC123")  // "ABC123TF"
isValid := base32.ValidateChecksum(withChecksum)      // true
checksum, err := base32.CalculateChecksum("ABC123")   // "TF"

// Normalization
normalized := base32.NormalizeBase32("abc-def")       // "ABCDEF"
normalized = base32.NormalizeBase32("1O 2I")          // "1021" (O→0, I→1)
```

**Key Features:**
- Human-readable Crockford Base32 alphabet (excludes I, L, O, U)
- Auto-correction: I→1, L→1, O→0
- CRC-10 checksum: 99.9%+ error detection
- Case-insensitive decoding
- URL-safe output

## Overview

The `base32` package provides:
- **Crockford Base32 Encoding**: Human-friendly encoding with automatic error correction
- **CRC-10 Checksums**: Industry-grade error detection (2 characters overhead)
- **Normalization**: Automatic handling of common input errors
- **Fixed and Compact Modes**: Choose between consistent length or minimal characters

## Running the Examples

To run the examples, use the following command from the repository root:

```bash
go run ./examples/base32
```

A second, more compact demo lives in `base32/examples` behind the `example` build tag:

```bash
go run -tags=example ./base32/examples
```

This will demonstrate:
1. Basic encoding and decoding operations
2. Checksum generation and validation
3. URL shortener implementation
4. Order/Transaction ID generation
5. License key generation
6. Voucher/Coupon code generation
7. IoT Device ID generation
8. Error correction and normalization
9. Comprehensive error detection

## Example Descriptions

The [example.go](example.go) file demonstrates several practical use cases:

### 1. Basic Encoding and Decoding

Shows fundamental encoding operations:

```go
// Fixed-length encoding
encoded, _ := base32.EncodeBase32(12345, 8)  // "00000C1S"

// Compact encoding (minimum characters)
compact := base32.EncodeBase32Compact(12345)  // "C1S"

// Case-insensitive decoding
value, _ := base32.DecodeBase32("c1s")  // 12345
```

### 2. Checksum Operations

Demonstrates all checksum functions:

```go
checksum, _ := base32.CalculateChecksum("ABC123")   // "TF"
withChecksum, _ := base32.AppendChecksum("ABC123")  // "ABC123TF"
isValid := base32.ValidateChecksum(withChecksum)    // true
extracted := base32.ExtractChecksum(withChecksum)   // "TF"
stripped := base32.StripChecksum(withChecksum)      // "ABC123"
```

### 3. URL Shortener

Complete URL shortener implementation:

```go
// Database ID to short code
databaseID := uint64(123456789)
shortCode := base32.EncodeBase32Compact(databaseID)
url := "https://short.url/" + shortCode  // "https://short.url/3NQK8N"

// Decode back to database ID
decoded, _ := base32.DecodeBase32(shortCode)  // 123456789
```

### 4. Order/Transaction IDs

Generate timestamped order IDs with sequences:

```go
timestamp := uint64(time.Now().Unix())
sequence := uint64(12345)

timeCode, _ := base32.EncodeBase32(timestamp, 8)
seqCode, _ := base32.EncodeBase32(sequence, 4)  // "0C1S"

// AppendChecksum normalizes its input: dashes removed, lookalikes
// corrected ("ORD" contains O → 0), then the checksum is appended.
orderID, _ := base32.AppendChecksum("ORD-" + timeCode + "-" + seqCode)
// "0RD" + timeCode + seqCode + 2-char checksum
```

### 5. License Key Generation

Create verifiable license keys:

```go
productID := uint64(42)
customerID := uint64(789)
expiryDate := uint64(20251231)

product, _ := base32.EncodeBase32(productID, 2)    // "1A"
customer, _ := base32.EncodeBase32(customerID, 4)  // "00RN"
expiry, _ := base32.EncodeBase32(expiryDate, 6)    // "0KA0JZ"

// Dashes are fine: AppendChecksum normalizes them away first
licenseKey, _ := base32.AppendChecksum(product + "-" + customer + "-" + expiry)
// "1A00RN0KA0JZE8" — includes automatic error detection
```

### 6. Voucher/Coupon Codes

Generate short, typeable voucher codes:

```go
voucherID := uint64(9999)
code, _ := base32.EncodeBase32(voucherID, 4)        // "09RF"
codeWithChecksum, _ := base32.AppendChecksum(code)  // "09RFZB"
// Display as "09-RF-ZB" (easy to type, error-correcting)
```

### 7. IoT Device IDs

Create compact device identifiers:

```go
deviceSerial := uint64(123456)
deviceID := base32.EncodeBase32Compact(deviceSerial)      // "3RJ0"
deviceIDWithChecksum, _ := base32.AppendChecksum(deviceID) // "3RJ0N1"
// Display as "DEV-3RJ0N1"
```

### 8. Error Correction and Normalization

Demonstrates automatic error correction:

```go
// Normalizes input automatically
base32.NormalizeBase32("abc-def")  // "ABCDEF" (uppercase, no dashes)
base32.NormalizeBase32("1O 2I")    // "1021" (O→0, I→1, no spaces)
base32.NormalizeBase32("ABC DEF")  // "ABCDEF" (uppercase, no spaces)

// Character validation
base32.IsValidBase32Char('A')  // true
base32.IsValidBase32Char('O')  // true (auto-corrected to '0')
base32.IsValidBase32Char('U')  // false (excluded from alphabet)
```

### 9. Checksum Validation and Error Detection

Shows error detection capabilities:

```go
validID, _ := base32.AppendChecksum("TEST123")  // "TEST123WB"
checksum := base32.ExtractChecksum(validID)     // "WB"

// Detects single character errors (100%)
corrupted := "XEST123" + checksum  // T→X
base32.ValidateChecksum(corrupted)  // false - detected!

// Detects transpositions (99.9%+)
transposed := "ETST123" + checksum  // TE→ET
base32.ValidateChecksum(transposed)  // false - detected!

// Detects wrong checksums
wrongChecksum := "TEST12300"
base32.ValidateChecksum(wrongChecksum)  // false - detected!
```

## Crockford Base32 Alphabet

```
0 1 2 3 4 5 6 7 8 9 A B C D E F G H J K M N P Q R S T V W X Y Z
```

**Excluded characters:**
- `I` - Looks like `1` (auto-corrected to `1`)
- `L` - Looks like `1` (auto-corrected to `1`)
- `O` - Looks like `0` (auto-corrected to `0`)
- `U` - Could be confused with `V`

This design minimizes human transcription errors.

## Error Detection

The CRC-10 checksum provides excellent error detection:

| Error Type | Detection Rate |
|------------|----------------|
| Single character error | 100% |
| Transposition (AB→BA) | 99.9%+ |
| Double errors | 99.9%+ |
| Insertion/deletion | High |

### How It Works

The examples demonstrate real error detection:
- Single character changes are always detected
- Character swaps (transpositions) are detected with high probability
- Wrong checksums are immediately identified
- Only 2 characters of overhead for robust protection

## Use Cases

### URL Shorteners
Convert database IDs to short, shareable links:
- Database ID 123456789 → `3NQK8N`
- Compact, URL-safe representation

### Order/Transaction IDs
Create timestamped, verifiable order identifiers:
- Includes timestamp and sequence number
- Checksum prevents typos in order lookup

### License Keys
Generate product licenses with validation:
- Product ID + Customer ID + Expiry Date
- Built-in error detection

### Voucher/Coupon Codes
Short codes for promotions:
- Human-friendly (no confusing characters)
- Error-correcting (auto-fixes common mistakes)
- Easy to type and read

### IoT Device IDs
Compact identifiers for devices:
- Serial number → Short device ID
- Includes checksum for validation

## Best Practices

### 1. Always Use Checksums for User-Facing IDs

```go
// ✓ Good - includes error detection
id, err := base32.AppendChecksum(data)

// ✗ Avoid - no error detection
id := data
```

### 2. Normalize User Input

```go
userInput := "ab-cd-ef"
normalized := base32.NormalizeBase32(userInput)  // "ABCDEF"
```

### 3. Use Fixed-Length for Databases

```go
// Consistent length for indexing
id := base32.EncodeBase32(value, 10)
```

### 4. Use Compact Encoding for URLs

```go
// Shorter URLs
shortCode := base32.EncodeBase32Compact(value)
```

### 5. Validate Before Processing

```go
if base32.ValidateChecksum(userInput) {
    // Process valid input
} else {
    // Handle invalid input
}
```

## Performance

Benchmarks on modern hardware:

```
BenchmarkEncodeBase32-8          50000000    25.3 ns/op
BenchmarkDecodeBase32-8          30000000    45.2 ns/op
BenchmarkCalculateChecksum-8     10000000   125.0 ns/op
BenchmarkValidateChecksum-8       8000000   160.0 ns/op
```

All operations are highly optimized for production use.

## API Reference

### Encoding Functions

- `EncodeBase32(value uint64, length int) (string, error)` - Fixed-length encoding
- `EncodeBase32Compact(value uint64) string` - Minimum character encoding

### Decoding Functions

- `DecodeBase32(encoded string) (uint64, error)` - Decode to uint64

### Checksum Functions

- `CalculateChecksum(data string) (string, error)` - Compute CRC-10 checksum
- `AppendChecksum(data string) (string, error)` - Add checksum to data (input normalized first)
- `ValidateChecksum(input string) bool` - Verify checksum (input normalized first)
- `StripChecksum(input string) string` - Remove checksum (last 2 chars)
- `ExtractChecksum(input string) string` - Get checksum (last 2 chars)

### Utility Functions

- `NormalizeBase32(input string) string` - Normalize and correct input
- `IsValidBase32Char(c rune) bool` - Check character validity

## Further Reading

- [Base32 Package Documentation](https://github.com/jasoet/pkg/tree/main/base32)
- [Crockford Base32 Specification](https://www.crockford.com/base32.html)
- [Main pkg Repository](https://github.com/jasoet/pkg)
