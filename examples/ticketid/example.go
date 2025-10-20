// Package main demonstrates comprehensive usage of the ticketid package.
//
// This example shows:
//   - Basic ticket ID generation and decoding
//   - Event ticketing system scenarios
//   - Error handling and validation
//   - Formatting and component extraction
//
// Run with: go run ./examples/ticketid
package main

import (
	"fmt"
	"time"

	"github.com/jasoet/pkg/v2/ticketid"
)

func main() {
	fmt.Println("=== TicketID Package Examples ===\n")

	// Example 1: Basic Generation
	basicGeneration()

	// Example 2: Decoding and Validation
	decodingAndValidation()

	// Example 3: Event Ticketing System
	eventTicketingSystem()

	// Example 4: Error Handling
	errorHandling()

	// Example 5: Formatting and Components
	formattingAndComponents()

	// Example 6: Batch Generation
	batchGeneration()
}

// Example 1: Basic ticket ID generation
func basicGeneration() {
	fmt.Println("--- Example 1: Basic Generation ---")

	// Generate a ticket with numeric IDs
	ticket1, err := ticketid.Generate(
		"12345",                                       // Event ID (numeric)
		time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC), // Event Date
		"100",                                         // Category ID (numeric)
		"500",                                         // Seat ID (numeric)
		ticketid.GenerateSequence(),                   // Auto-generated sequence
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Ticket 1 (numeric IDs): %s\n", ticket1)
	fmt.Printf("Formatted:              %s\n", ticketid.Format(ticket1))

	// Generate a ticket with string IDs
	ticket2, err := ticketid.Generate(
		"EVT001",                                    // Event ID (string)
		time.Date(2025, 7, 4, 0, 0, 0, 0, time.UTC), // Event Date
		"VIP",                                       // Category (string)
		"A123",                                      // Seat ID (string)
		12345,                                       // Explicit sequence
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Ticket 2 (string IDs):  %s\n", ticket2)
	fmt.Printf("Formatted:              %s\n\n", ticketid.Format(ticket2))
}

// Example 2: Decoding and validating ticket IDs
func decodingAndValidation() {
	fmt.Println("--- Example 2: Decoding and Validation ---")

	// Generate a ticket
	ticketID, _ := ticketid.Generate(
		"CONCERT2025",
		time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC),
		"GOLD",
		"SEAT-A42",
		99999,
	)

	fmt.Printf("Generated Ticket: %s\n", ticketID)

	// Validate format
	if ticketid.IsValidTicketIDFormat(ticketID) {
		fmt.Println("✓ Ticket ID format is valid")
	}

	// Decode the ticket
	decoded, err := ticketid.DecodeTicketID(ticketID)
	if err != nil {
		fmt.Printf("Error decoding: %v\n", err)
		return
	}

	fmt.Println("\nDecoded Components:")
	fmt.Printf("  Event ID:    %s\n", decoded.EventID)
	fmt.Printf("  Event Date:  %s\n", decoded.EventDate.Format("2006-01-02"))
	fmt.Printf("  Category:    %s\n", decoded.CategoryID)
	fmt.Printf("  Seat ID:     %s\n", decoded.SeatID)
	fmt.Printf("  Sequence:    %d\n", decoded.Sequence)
	fmt.Printf("  Checksum:    %s\n\n", decoded.Checksum)

	// Test with formatted ticket (with dashes)
	formattedTicket := ticketid.Format(ticketID)
	decodedFromFormatted, err := ticketid.DecodeTicketID(formattedTicket)
	if err != nil {
		fmt.Printf("Error decoding formatted: %v\n", err)
		return
	}
	fmt.Printf("Formatted ticket decoded successfully: %s\n\n", decodedFromFormatted.EventID)
}

// Example 3: Complete event ticketing system scenario
func eventTicketingSystem() {
	fmt.Println("--- Example 3: Event Ticketing System ---")

	// Define event details
	eventName := "SUMMER2025"
	eventDate := time.Date(2025, 6, 21, 0, 0, 0, 0, time.UTC)

	// Different ticket categories
	categories := map[string]string{
		"VIP":      "VIP",
		"PREMIUM":  "PREMIUM",
		"GENERAL":  "GENERAL",
		"STUDENT":  "STUDENT",
	}

	fmt.Printf("Event: %s on %s\n\n", eventName, eventDate.Format("2006-01-02"))

	// Generate tickets for different categories and seats
	tickets := make(map[string]string)

	// VIP Section
	vipTicket, _ := ticketid.Generate(eventName, eventDate, categories["VIP"], "VIP-A1", 1001)
	tickets["VIP-A1"] = vipTicket
	fmt.Printf("VIP Ticket (A1):      %s\n", ticketid.Format(vipTicket))

	// Premium Section
	premiumTicket, _ := ticketid.Generate(eventName, eventDate, categories["PREMIUM"], "PREM-B10", 2001)
	tickets["PREM-B10"] = premiumTicket
	fmt.Printf("Premium Ticket (B10): %s\n", ticketid.Format(premiumTicket))

	// General Admission
	generalTicket, _ := ticketid.Generate(eventName, eventDate, categories["GENERAL"], "GEN-C50", 3001)
	tickets["GEN-C50"] = generalTicket
	fmt.Printf("General Ticket (C50): %s\n", ticketid.Format(generalTicket))

	// Student Ticket
	studentTicket, _ := ticketid.Generate(eventName, eventDate, categories["STUDENT"], "STU-D25", 4001)
	tickets["STU-D25"] = studentTicket
	fmt.Printf("Student Ticket (D25): %s\n\n", ticketid.Format(studentTicket))

	// Simulate ticket validation at entrance
	fmt.Println("Ticket Validation at Entrance:")
	for seat, ticket := range tickets {
		if ticketid.IsValidTicketIDFormat(ticket) {
			decoded, _ := ticketid.DecodeTicketID(ticket)
			fmt.Printf("✓ %s - Valid - Category: %s\n", seat, decoded.CategoryID)
		} else {
			fmt.Printf("✗ %s - Invalid ticket\n", seat)
		}
	}
	fmt.Println()
}

// Example 4: Error handling scenarios
func errorHandling() {
	fmt.Println("--- Example 4: Error Handling ---")

	// Error 1: Empty event ID
	_, err := ticketid.Generate("", time.Now(), "CAT", "SEAT", 1)
	if err != nil {
		fmt.Printf("✓ Caught error - Empty event ID: %v\n", err)
	}

	// Error 2: Empty category
	_, err = ticketid.Generate("EVT", time.Now(), "", "SEAT", 1)
	if err != nil {
		fmt.Printf("✓ Caught error - Empty category: %v\n", err)
	}

	// Error 3: Negative sequence
	_, err = ticketid.Generate("EVT", time.Now(), "CAT", "SEAT", -1)
	if err != nil {
		fmt.Printf("✓ Caught error - Negative sequence: %v\n", err)
	}

	// Error 4: Value out of range
	_, err = ticketid.Generate("33554432", time.Now(), "CAT", "SEAT", 1) // Event ID too large
	if err != nil {
		fmt.Printf("✓ Caught error - Event ID too large: %v\n", err)
	}

	// Error 5: Invalid ticket ID length
	_, err = ticketid.DecodeTicketID("SHORT")
	if err != nil {
		fmt.Printf("✓ Caught error - Invalid length: %v\n", err)
	}

	// Error 6: Invalid checksum
	validTicket, _ := ticketid.Generate("EVT", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), "CAT", "SEAT", 1)
	corruptedTicket := validTicket[:22] + "00" // Change checksum
	_, err = ticketid.DecodeTicketID(corruptedTicket)
	if err != nil {
		fmt.Printf("✓ Caught error - Invalid checksum: %v\n", err)
	}

	fmt.Println()
}

// Example 5: Formatting and component extraction
func formattingAndComponents() {
	fmt.Println("--- Example 5: Formatting and Components ---")

	// Generate a ticket
	ticket, _ := ticketid.Generate(
		"FESTIVAL",
		time.Date(2025, 9, 20, 0, 0, 0, 0, time.UTC),
		"BACKSTAGE",
		"B1",
		77777,
	)

	fmt.Printf("Raw Ticket:       %s\n", ticket)
	fmt.Printf("Formatted Ticket: %s\n\n", ticketid.Format(ticket))

	// Extract components without full decoding
	components := ticketid.ExtractComponents(ticket)
	if components != nil {
		fmt.Println("Extracted Components (raw):")
		fmt.Printf("  Event ID:   %s\n", components["eventID"])
		fmt.Printf("  Event Date: %s\n", components["eventDate"])
		fmt.Printf("  Category:   %s\n", components["category"])
		fmt.Printf("  Seat ID:    %s\n", components["seatID"])
		fmt.Printf("  Sequence:   %s\n", components["sequence"])
		fmt.Printf("  Checksum:   %s\n", components["checksum"])
	}

	// Remove dashes from formatted ticket
	formattedTicket := ticketid.Format(ticket)
	withoutDashes := ticketid.RemoveDashes(formattedTicket)
	fmt.Printf("\nWith dashes:    %s\n", formattedTicket)
	fmt.Printf("Without dashes: %s\n", withoutDashes)
	fmt.Printf("Matches raw:    %v\n\n", ticket == withoutDashes)
}

// Example 6: Batch ticket generation
func batchGeneration() {
	fmt.Println("--- Example 6: Batch Generation ---")

	eventDate := time.Date(2025, 10, 31, 0, 0, 0, 0, time.UTC)
	fmt.Println("Generating 10 tickets for Halloween Concert 2025...")
	fmt.Println()

	tickets := make([]string, 10)
	for i := 0; i < 10; i++ {
		seatID := fmt.Sprintf("SEAT-%03d", i+1)
		ticket, err := ticketid.Generate(
			"HALLOWEEN2025",
			eventDate,
			"GA", // General Admission
			seatID,
			ticketid.GenerateSequence(), // Unique sequence
		)
		if err != nil {
			fmt.Printf("Error generating ticket %d: %v\n", i+1, err)
			continue
		}
		tickets[i] = ticket
		fmt.Printf("Ticket %2d (%s): %s\n", i+1, seatID, ticketid.Format(ticket))
	}

	// Verify all tickets are unique
	uniqueTickets := make(map[string]bool)
	for _, ticket := range tickets {
		uniqueTickets[ticket] = true
	}

	fmt.Printf("\nGenerated %d unique tickets\n", len(uniqueTickets))
	if len(uniqueTickets) == len(tickets) {
		fmt.Println("✓ All tickets are unique!")
	}
}
