package base32_test

import (
	"fmt"

	"github.com/jasoet/pkg/v3/base32"
)

// All // Output: blocks below are verified by `go test` against real
// function output.

func ExampleEncodeBase32() {
	id, _ := base32.EncodeBase32(12345, 8)
	fmt.Println(id)
	// Output: 00000C1S
}

func ExampleEncodeBase32Compact() {
	fmt.Println(base32.EncodeBase32Compact(12345))
	fmt.Println(base32.EncodeBase32Compact(123456789))
	// Output:
	// C1S
	// 3NQK8N
}

func ExampleDecodeBase32() {
	value, _ := base32.DecodeBase32("C1S")
	fmt.Println(value)
	// Output: 12345
}

func ExampleCalculateChecksum() {
	checksum, _ := base32.CalculateChecksum("ABC123")
	fmt.Println(checksum)
	// Output: TF
}

func ExampleAppendChecksum() {
	withChecksum, _ := base32.AppendChecksum("ABC123")
	fmt.Println(withChecksum)
	// Output: ABC123TF
}

// AppendChecksum normalizes its input, so dashed/lowercase identifiers work.
func ExampleAppendChecksum_normalized() {
	withChecksum, _ := base32.AppendChecksum("0000-c1p9")
	fmt.Println(withChecksum)
	// Output: 0000C1P9Q0
}

func ExampleValidateChecksum() {
	fmt.Println(base32.ValidateChecksum("ABC123TF"))
	fmt.Println(base32.ValidateChecksum("0000-c1p9-q0")) // normalized before validation
	fmt.Println(base32.ValidateChecksum("ABC123ZZ"))
	// Output:
	// true
	// true
	// false
}

func ExampleNormalizeBase32() {
	fmt.Println(base32.NormalizeBase32("abc-def"))
	fmt.Println(base32.NormalizeBase32("1O 2I"))
	// Output:
	// ABCDEF
	// 1021
}
