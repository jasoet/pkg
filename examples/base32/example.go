// Package main demonstrates comprehensive usage of the base32 package.
//
// This example shows:
//   - Crockford Base32 encoding and decoding
//   - CRC-10 checksum operations
//   - Real-world use cases (URL shorteners, order IDs, license keys, etc.)
//   - Error correction and normalization
//
// Run with: go run ./examples/base32
package main

import (
	"fmt"
	"time"

	"github.com/jasoet/pkg/v2/base32"
)

func main() {
	fmt.Println("=== Base32 Package Examples ===")

	// Example 1: Basic Encoding and Decoding
	basicEncodingDecoding()

	// Example 2: Checksum Operations
	checksumOperations()

	// Example 3: URL Shortener
	urlShortener()

	// Example 4: Order/Transaction IDs
	orderTransactionIDs()

	// Example 5: License Key Generation
	licenseKeyGeneration()

	// Example 6: Voucher/Coupon Codes
	voucherCouponCodes()

	// Example 7: IoT Device IDs
	iotDeviceIDs()

	// Example 8: Error Correction and Normalization
	errorCorrectionNormalization()

	// Example 9: Checksum Validation and Error Detection
	checksumValidation()
}

// Example 1: Basic encoding and decoding
func basicEncodingDecoding() {
	fmt.Println("--- Example 1: Basic Encoding and Decoding ---")

	// Fixed-length encoding
	value1 := uint64(12345)
	encoded1 := base32.EncodeBase32(value1, 8)
	fmt.Printf("Encode %d with length 8: %s\n", value1, encoded1)

	// Compact encoding (minimum characters)
	encoded2 := base32.EncodeBase32Compact(value1)
	fmt.Printf("Encode %d compact:       %s\n", value1, encoded2)

	// Decoding
	decoded, err := base32.DecodeBase32(encoded2)
	if err != nil {
		fmt.Printf("Error decoding: %v\n", err)
		return
	}
	fmt.Printf("Decode '%s':             %d\n", encoded2, decoded)

	// Case-insensitive decoding
	decodedLower, _ := base32.DecodeBase32("c1p9")
	fmt.Printf("Decode 'c1p9' (lower):   %d\n\n", decodedLower)
}

// Example 2: Checksum operations
func checksumOperations() {
	fmt.Println("--- Example 2: Checksum Operations ---")

	data := "ABC123"

	// Calculate checksum
	checksum := base32.CalculateChecksum(data)
	fmt.Printf("Data:               %s\n", data)
	fmt.Printf("Checksum:           %s\n", checksum)

	// Append checksum
	withChecksum := base32.AppendChecksum(data)
	fmt.Printf("With checksum:      %s\n", withChecksum)

	// Validate checksum
	isValid := base32.ValidateChecksum(withChecksum)
	fmt.Printf("Is valid:           %v\n", isValid)

	// Extract components
	extractedChecksum := base32.ExtractChecksum(withChecksum)
	strippedData := base32.StripChecksum(withChecksum)
	fmt.Printf("Extracted checksum: %s\n", extractedChecksum)
	fmt.Printf("Stripped data:      %s\n\n", strippedData)
}

// Example 3: URL shortener use case
func urlShortener() {
	fmt.Println("--- Example 3: URL Shortener ---")

	// Simulate database IDs
	urlIDs := []uint64{1, 100, 10000, 123456789, 999999999}

	fmt.Println("Database ID → Short Code")
	for _, id := range urlIDs {
		shortCode := base32.EncodeBase32Compact(id)
		fmt.Printf("%10d → https://short.url/%s\n", id, shortCode)
	}

	// Decode a short code back to database ID
	shortCode := "3QTYY1"
	decodedID, _ := base32.DecodeBase32(shortCode)
	fmt.Printf("\nDecode '%s' → Database ID: %d\n\n", shortCode, decodedID)
}

// Example 4: Order/Transaction ID generation
func orderTransactionIDs() {
	fmt.Println("--- Example 4: Order/Transaction IDs ---")

	// Generate order ID with timestamp and sequence
	timestamp := uint64(time.Now().Unix())
	sequence := uint64(12345)

	timeCode := base32.EncodeBase32(timestamp, 8)
	seqCode := base32.EncodeBase32(sequence, 4)

	// Combine and add checksum
	orderIDData := "ORD-" + timeCode + "-" + seqCode
	orderID := base32.AppendChecksum(orderIDData)

	fmt.Printf("Timestamp:  %d\n", timestamp)
	fmt.Printf("Sequence:   %d\n", sequence)
	fmt.Printf("Order ID:   %s\n", orderID)

	// Validate the order ID
	if base32.ValidateChecksum(orderID) {
		fmt.Println("✓ Order ID checksum is valid")

		// Extract components
		orderData := base32.StripChecksum(orderID)
		fmt.Printf("Order Data: %s\n", orderData)
	}

	fmt.Println()
}

// Example 5: License key generation
func licenseKeyGeneration() {
	fmt.Println("--- Example 5: License Key Generation ---")

	productID := uint64(42)
	customerID := uint64(789)
	expiryDate := uint64(20251231) // YYYYMMDD

	product := base32.EncodeBase32(productID, 2)
	customer := base32.EncodeBase32(customerID, 4)
	expiry := base32.EncodeBase32(expiryDate, 6)

	licenseData := product + "-" + customer + "-" + expiry
	licenseKey := base32.AppendChecksum(licenseData)

	fmt.Printf("Product ID:    %d\n", productID)
	fmt.Printf("Customer ID:   %d\n", customerID)
	fmt.Printf("Expiry Date:   %d\n", expiryDate)
	fmt.Printf("License Key:   %s\n", licenseKey)

	// Format for display (add dashes every 4 chars)
	formatted := formatLicenseKey(licenseKey)
	fmt.Printf("Formatted:     %s\n", formatted)

	// Validate
	if base32.ValidateChecksum(licenseKey) {
		fmt.Println("✓ License key is valid")
	}

	fmt.Println()
}

// Example 6: Voucher/Coupon code generation
func voucherCouponCodes() {
	fmt.Println("--- Example 6: Voucher/Coupon Codes ---")

	// Generate voucher codes
	voucherIDs := []uint64{1, 99, 999, 9999}

	fmt.Println("Voucher ID → Code")
	for _, id := range voucherIDs {
		code := base32.EncodeBase32(id, 4)
		codeWithChecksum := base32.AppendChecksum(code)

		// Format with dash for readability
		formatted := code[:2] + "-" + code[2:] + "-" + base32.ExtractChecksum(codeWithChecksum)

		fmt.Printf("%5d → %s (checksum: %s)\n",
			id,
			formatted,
			base32.ExtractChecksum(codeWithChecksum))
	}

	// Validate a voucher code
	fmt.Println("\nVoucher Validation:")
	testCode := "09-ZZ"
	fullCode := testCode + base32.CalculateChecksum(base32.NormalizeBase32(testCode))

	if base32.ValidateChecksum(base32.NormalizeBase32(fullCode)) {
		fmt.Printf("✓ Code '%s' is valid\n", testCode)
	}

	fmt.Println()
}

// Example 7: IoT Device ID generation
func iotDeviceIDs() {
	fmt.Println("--- Example 7: IoT Device IDs ---")

	// Generate device IDs
	deviceSerials := []uint64{123456, 234567, 345678, 456789, 567890}

	fmt.Println("Serial Number → Device ID")
	for _, serial := range deviceSerials {
		deviceID := base32.EncodeBase32Compact(serial)
		deviceIDWithChecksum := base32.AppendChecksum(deviceID)

		fmt.Printf("%6d → DEV-%s (with checksum: DEV-%s)\n",
			serial,
			deviceID,
			deviceIDWithChecksum)
	}

	// Validate a device ID
	fmt.Println("\nDevice ID Validation:")
	testDeviceID := "DEV-3QTY01XY"
	devicePart := testDeviceID[4:] // Remove "DEV-" prefix

	if base32.ValidateChecksum(devicePart) {
		fmt.Printf("✓ Device ID '%s' is valid\n", testDeviceID)

		// Decode to get original serial
		serial, _ := base32.DecodeBase32(base32.StripChecksum(devicePart))
		fmt.Printf("  Serial Number: %d\n", serial)
	}

	fmt.Println()
}

// Example 8: Error correction and normalization
func errorCorrectionNormalization() {
	fmt.Println("--- Example 8: Error Correction and Normalization ---")

	// Common input errors
	inputs := []string{
		"abc-def",      // With dashes
		"ABC DEF",      // With spaces
		"abc def",      // Lowercase with spaces
		"1O 2I",        // Contains O (→0) and I (→1)
		"1L 2O",        // Contains L (→1) and O (→0)
		"ABCD-EFGH-IJ", // Mixed case with dashes and confusing chars
	}

	fmt.Println("Input → Normalized")
	for _, input := range inputs {
		normalized := base32.NormalizeBase32(input)
		fmt.Printf("%-20s → %s\n", input, normalized)
	}

	// Character validation
	fmt.Println("\nCharacter Validation:")
	chars := []rune{'A', 'Z', '0', '9', 'O', 'I', 'L', 'U', '@', ' '}
	for _, c := range chars {
		isValid := base32.IsValidBase32Char(c)
		status := "✗"
		if isValid {
			status = "✓"
		}
		fmt.Printf("%s '%c' is valid\n", status, c)
	}

	fmt.Println()
}

// Example 9: Checksum validation and error detection
func checksumValidation() {
	fmt.Println("--- Example 9: Checksum Validation and Error Detection ---")

	// Create a valid ID
	validID := base32.AppendChecksum("TEST123")
	fmt.Printf("Valid ID: %s\n", validID)

	// Test 1: Valid checksum
	fmt.Println("\nTest 1: Valid Checksum")
	if base32.ValidateChecksum(validID) {
		fmt.Println("✓ Checksum is valid")
	}

	// Test 2: Single character error
	fmt.Println("\nTest 2: Single Character Error (T→X)")
	corrupted := "XEST123" + base32.ExtractChecksum(validID)
	if !base32.ValidateChecksum(corrupted) {
		fmt.Println("✓ Error detected!")
	}

	// Test 3: Transposition error
	fmt.Println("\nTest 3: Transposition Error (TE→ET)")
	transposed := "ETST123" + base32.ExtractChecksum(validID)
	if !base32.ValidateChecksum(transposed) {
		fmt.Println("✓ Transposition detected!")
	}

	// Test 4: Wrong checksum
	fmt.Println("\nTest 4: Wrong Checksum")
	wrongChecksum := "TEST12300" // Wrong checksum
	if !base32.ValidateChecksum(wrongChecksum) {
		fmt.Println("✓ Invalid checksum detected!")
	}

	// Test 5: Error detection rate demonstration
	fmt.Println("\nTest 5: Error Detection Demonstration")
	testData := "ABCD1234"
	validData := base32.AppendChecksum(testData)

	fmt.Printf("Original: %s\n", validData)
	fmt.Println("\nTesting various corruptions:")

	corruptions := map[string]string{
		"Change A→X":        "XBCD1234" + base32.ExtractChecksum(validData),
		"Change 1→2":        "ABCD2234" + base32.ExtractChecksum(validData),
		"Swap AB→BA":        "BACD1234" + base32.ExtractChecksum(validData),
		"Delete character":  "ABCD123" + base32.ExtractChecksum(validData),
		"Add character":     "ABCD12345" + base32.ExtractChecksum(validData),
		"Change checksum":   testData + "00",
	}

	for desc, corrupted := range corruptions {
		detected := !base32.ValidateChecksum(corrupted)
		status := "✗ Not detected"
		if detected {
			status = "✓ Detected"
		}
		fmt.Printf("  %-20s: %s\n", desc, status)
	}

	fmt.Println()
}

// Helper function to format license key with dashes
func formatLicenseKey(key string) string {
	if len(key) < 4 {
		return key
	}

	var formatted string
	for i, c := range key {
		if i > 0 && i%4 == 0 && i < len(key) {
			formatted += "-"
		}
		formatted += string(c)
	}
	return formatted
}
