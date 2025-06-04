package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// GreetingInput is the input for the Greeting activity
type GreetingInput struct {
	Name string
}

// GreetingResult is the result of the Greeting activity
type GreetingResult struct {
	Greeting string
	Time     time.Time
}

// Greeting is a simple activity that generates a greeting message
func Greeting(ctx context.Context, input GreetingInput) (GreetingResult, error) {
	logger := log.With().
		Str("activity", "Greeting").
		Str("name", input.Name).
		Logger()

	logger.Info().Msg("Executing Greeting activity")

	// Validate input
	if input.Name == "" {
		input.Name = "World"
		logger.Info().Msg("No name provided, using default")
	}

	// Simulate some work
	time.Sleep(500 * time.Millisecond)

	// Create the greeting
	greeting := fmt.Sprintf("Hello, %s!", input.Name)
	result := GreetingResult{
		Greeting: greeting,
		Time:     time.Now(),
	}

	logger.Info().
		Str("greeting", result.Greeting).
		Time("time", result.Time).
		Msg("Greeting activity completed")

	return result, nil
}

// ProcessDataInput is the input for the ProcessData activity
type ProcessDataInput struct {
	Data     string
	Multiply int
}

// ProcessDataResult is the result of the ProcessData activity
type ProcessDataResult struct {
	ProcessedData string
	Count         int
}

// ProcessData is an activity that processes some data
func ProcessData(ctx context.Context, input ProcessDataInput) (ProcessDataResult, error) {
	logger := log.With().
		Str("activity", "ProcessData").
		Str("data", input.Data).
		Int("multiply", input.Multiply).
		Logger()

	logger.Info().Msg("Executing ProcessData activity")

	// Validate input
	if input.Data == "" {
		err := fmt.Errorf("no data provided")
		logger.Error().Err(err).Msg("Invalid input")
		return ProcessDataResult{}, err
	}

	if input.Multiply <= 0 {
		input.Multiply = 1
		logger.Info().Msg("Invalid multiply value, using default")
	}

	// Simulate some work
	time.Sleep(1 * time.Second)

	// Process the data
	processedData := ""
	for i := 0; i < input.Multiply; i++ {
		processedData += input.Data
	}

	result := ProcessDataResult{
		ProcessedData: processedData,
		Count:         len(processedData),
	}

	logger.Info().
		Str("processedData", result.ProcessedData).
		Int("count", result.Count).
		Msg("ProcessData activity completed")

	return result, nil
}

// FetchExternalDataInput is the input for the FetchExternalData activity
type FetchExternalDataInput struct {
	URL string
}

// FetchExternalDataResult is the result of the FetchExternalData activity
type FetchExternalDataResult struct {
	Data       string
	StatusCode int
	FetchedAt  time.Time
}

// FetchExternalData simulates fetching data from an external service
func FetchExternalData(ctx context.Context, input FetchExternalDataInput) (FetchExternalDataResult, error) {
	logger := log.With().
		Str("activity", "FetchExternalData").
		Str("url", input.URL).
		Logger()

	logger.Info().Msg("Executing FetchExternalData activity")

	// Validate input
	if input.URL == "" {
		err := fmt.Errorf("no URL provided")
		logger.Error().Err(err).Msg("Invalid input")
		return FetchExternalDataResult{}, err
	}

	// Simulate fetching data from an external service
	logger.Info().Msg("Connecting to external service...")
	time.Sleep(2 * time.Second)

	// Simulate different responses based on the URL
	var result FetchExternalDataResult
	result.FetchedAt = time.Now()

	if input.URL == "https://example.com/error" {
		err := fmt.Errorf("external service returned an error")
		logger.Error().Err(err).Msg("Failed to fetch data")
		return FetchExternalDataResult{}, err
	} else if input.URL == "https://example.com/timeout" {
		// Simulate a timeout
		time.Sleep(3 * time.Second)
		err := fmt.Errorf("request timed out")
		logger.Error().Err(err).Msg("Request timed out")
		return FetchExternalDataResult{}, err
	} else {
		// Simulate successful response
		result.Data = fmt.Sprintf("Data from %s", input.URL)
		result.StatusCode = 200
		logger.Info().
			Str("data", result.Data).
			Int("statusCode", result.StatusCode).
			Time("fetchedAt", result.FetchedAt).
			Msg("Successfully fetched data")
	}

	return result, nil
}
