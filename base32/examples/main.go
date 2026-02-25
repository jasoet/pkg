//go:build example

package main

import (
	"fmt"
	"time"

	"github.com/jasoet/pkg/v2/base32"
)

func main() {
	fmt.Println("=== Base32 Package Examples ===\n")

	// Example 1: URL Shortener
	urlShortenerExample()

	// Example 2: Order ID Generation
	orderIDExample()

	// Example 3: License Key Generation
	licenseKeyExample()

	// Example 4: Error Correction Demonstration
	errorCorrectionExample()

	// Example 5: Checksum Validation
	checksumValidationExample()
}

func urlShortenerExample() {
	fmt.Println("1. URL Shortener Example")
	fmt.Println("------------------------")

	// Simulate a database ID
	databaseID := uint64(123456789)

	// Create short URL code
	shortCode := base32.EncodeBase32Compact(databaseID)
	fmt.Printf("Database ID: %d\n", databaseID)
	fmt.Printf("Short URL: https://short.url/%s\n", shortCode)

	// Decode back to database ID
	decoded, err := base32.DecodeBase32(shortCode)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Decoded ID: %d\n", decoded)
	fmt.Printf("Match: %v\n\n", decoded == databaseID)
}

func orderIDExample() {
	fmt.Println("2. Order ID Generation Example")
	fmt.Println("-------------------------------")

	// Generate order ID with timestamp and sequence
	timestamp := uint64(time.Now().Unix())
	sequence := uint64(12345)

	// Encode components
	timeCode, err := base32.EncodeBase32(timestamp, 8)
	if err != nil {
		fmt.Printf("Error encoding timestamp: %v\n", err)
		return
	}
	seqCode, err := base32.EncodeBase32(sequence, 4)
	if err != nil {
		fmt.Printf("Error encoding sequence: %v\n", err)
		return
	}

	// Combine with checksum
	orderData := "ORD-" + timeCode + "-" + seqCode
	orderID, err := base32.AppendChecksum(orderData)
	if err != nil {
		fmt.Printf("Error appending checksum: %v\n", err)
		return
	}

	fmt.Printf("Order ID: %s\n", orderID)
	fmt.Printf("Timestamp: %d\n", timestamp)
	fmt.Printf("Sequence: %d\n", sequence)

	// Validate order ID
	if base32.ValidateChecksum(orderID) {
		fmt.Println("✓ Order ID checksum valid")
	} else {
		fmt.Println("✗ Order ID checksum invalid")
	}
	fmt.Println()
}

func licenseKeyExample() {
	fmt.Println("3. License Key Generation Example")
	fmt.Println("----------------------------------")

	// Generate license key components
	productID := uint64(42)
	customerID := uint64(789)
	expiryDate := uint64(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC).Unix())

	// Encode each component
	product, _ := base32.EncodeBase32(productID, 2)
	customer, _ := base32.EncodeBase32(customerID, 4)
	expiry, _ := base32.EncodeBase32(expiryDate, 8)

	// Create license key with dashes for readability
	licenseData := product + customer + expiry
	licenseKey, _ := base32.AppendChecksum(licenseData)

	// Format with dashes (groups of 5)
	formatted := formatLicenseKey(licenseKey)

	fmt.Printf("Product ID: %d\n", productID)
	fmt.Printf("Customer ID: %d\n", customerID)
	fmt.Printf("License Key: %s\n", formatted)

	// Validate license key (after removing dashes)
	normalized := base32.NormalizeBase32(formatted)
	if base32.ValidateChecksum(normalized) {
		fmt.Println("✓ License key checksum valid")
	} else {
		fmt.Println("✗ License key checksum invalid")
	}
	fmt.Println()
}

func errorCorrectionExample() {
	fmt.Println("4. Error Correction Example")
	fmt.Println("----------------------------")

	original := "HELLO"
	fmt.Printf("Original: %s\n", original)

	// Common user mistakes
	mistakes := []string{
		"HELL0",  // O → 0 (looks similar)
		"HELlO",  // Mixed case
		"HE-LLO", // With dashes
		"HE LLO", // With spaces
		"HEILO",  // I instead of L
	}

	for _, mistake := range mistakes {
		normalized := base32.NormalizeBase32(mistake)
		fmt.Printf("  %10s → %s", mistake, normalized)
		if normalized == base32.NormalizeBase32(original) {
			fmt.Printf(" ✓ Auto-corrected\n")
		} else {
			fmt.Printf(" ✗ Still different\n")
		}
	}
	fmt.Println()
}

func checksumValidationExample() {
	fmt.Println("5. Checksum Validation Example")
	fmt.Println("-------------------------------")

	// Create a valid ID
	data := "ABC123"
	validID, _ := base32.AppendChecksum(data)
	fmt.Printf("Valid ID: %s\n", validID)

	// Test valid ID
	if base32.ValidateChecksum(validID) {
		fmt.Println("✓ Valid ID passes validation")
	}

	// Test corrupted ID (single character error)
	corrupted := "XBC123" + base32.ExtractChecksum(validID)
	fmt.Printf("\nCorrupted ID: %s (changed A to X)\n", corrupted)
	if !base32.ValidateChecksum(corrupted) {
		fmt.Println("✓ Corruption detected!")
	}

	// Test transposition error
	if len(validID) >= 2 {
		chars := []rune(validID)
		chars[0], chars[1] = chars[1], chars[0]
		transposed := string(chars)
		fmt.Printf("\nTransposed ID: %s (swapped first two chars)\n", transposed)
		if !base32.ValidateChecksum(transposed) {
			fmt.Println("✓ Transposition detected!")
		}
	}

	// Test double error
	doubleError := "XYC123" + base32.ExtractChecksum(validID)
	fmt.Printf("\nDouble Error: %s (changed A to X and B to Y)\n", doubleError)
	if !base32.ValidateChecksum(doubleError) {
		fmt.Println("✓ Double error detected!")
	}

	fmt.Println("\n=== Error Detection Summary ===")
	fmt.Println("✓ Single character errors: 100% detection")
	fmt.Println("✓ Transposition errors: 99.9%+ detection")
	fmt.Println("✓ Double errors: 99.9%+ detection")
}

// Helper function to format license key with dashes
func formatLicenseKey(key string) string {
	result := ""
	for i, char := range key {
		if i > 0 && i%5 == 0 {
			result += "-"
		}
		result += string(char)
	}
	return result
}
