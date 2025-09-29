//go:build examples

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	calculatorv1 "github.com/jasoet/pkg/grpc/examples/gen/calculator/v1"
)

func main() {
	serverAddr := "localhost:50051"
	if addr := os.Getenv("SERVER_ADDR"); addr != "" {
		serverAddr = addr
	}

	log.Printf("Connecting to calculator server at %s", serverAddr)

	// Connect to the server
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := calculatorv1.NewCalculatorServiceClient(conn)

	// Test unary RPCs
	testUnaryRPCs(client)

	// Test server streaming RPC
	testServerStreaming(client)

	// Test client streaming RPC
	testClientStreaming(client)

	// Test bidirectional streaming RPC
	testBidirectionalStreaming(client)

	log.Printf("All examples completed successfully!")
}

func testUnaryRPCs(client calculatorv1.CalculatorServiceClient) {
	fmt.Println("\n=== Testing Unary RPCs ===")

	// Test Add
	ctx := context.Background()
	addResp, err := client.Add(ctx, &calculatorv1.AddRequest{A: 10, B: 5})
	if err != nil {
		log.Printf("Add failed: %v", err)
	} else {
		fmt.Printf("10 + 5 = %.2f\n", addResp.Result)
	}

	// Test Subtract
	subResp, err := client.Subtract(ctx, &calculatorv1.SubtractRequest{A: 10, B: 3})
	if err != nil {
		log.Printf("Subtract failed: %v", err)
	} else {
		fmt.Printf("10 - 3 = %.2f\n", subResp.Result)
	}

	// Test Multiply
	mulResp, err := client.Multiply(ctx, &calculatorv1.MultiplyRequest{A: 4, B: 6})
	if err != nil {
		log.Printf("Multiply failed: %v", err)
	} else {
		fmt.Printf("4 * 6 = %.2f\n", mulResp.Result)
	}

	// Test Divide
	divResp, err := client.Divide(ctx, &calculatorv1.DivideRequest{A: 15, B: 3})
	if err != nil {
		log.Printf("Divide failed: %v", err)
	} else {
		fmt.Printf("15 / 3 = %.2f\n", divResp.Result)
	}

	// Test Divide by zero (should return error)
	_, err = client.Divide(ctx, &calculatorv1.DivideRequest{A: 10, B: 0})
	if err != nil {
		fmt.Printf("Division by zero correctly returned error: %v\n", err)
	}
}

func testServerStreaming(client calculatorv1.CalculatorServiceClient) {
	fmt.Println("\n=== Testing Server Streaming RPC (Factorial) ===")

	ctx := context.Background()
	stream, err := client.Factorial(ctx, &calculatorv1.FactorialRequest{Number: 5})
	if err != nil {
		log.Printf("Factorial failed: %v", err)
		return
	}

	fmt.Printf("Calculating factorial of 5:\n")
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Failed to receive from stream: %v", err)
			break
		}
		fmt.Printf("Step %d: %d\n", resp.Step, resp.Result)
	}
}

func testClientStreaming(client calculatorv1.CalculatorServiceClient) {
	fmt.Println("\n=== Testing Client Streaming RPC (Sum) ===")

	ctx := context.Background()
	stream, err := client.Sum(ctx)
	if err != nil {
		log.Printf("Sum failed: %v", err)
		return
	}

	numbers := []float64{1.5, 2.5, 3.5, 4.5, 5.5}
	fmt.Printf("Sending numbers to sum: %v\n", numbers)

	for _, num := range numbers {
		if err := stream.Send(&calculatorv1.SumRequest{Number: num}); err != nil {
			log.Printf("Failed to send number: %v", err)
			return
		}
		time.Sleep(100 * time.Millisecond) // Small delay to see the streaming effect
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Printf("Failed to receive sum: %v", err)
		return
	}

	fmt.Printf("Sum of all numbers: %.2f\n", resp.Total)
}

func testBidirectionalStreaming(client calculatorv1.CalculatorServiceClient) {
	fmt.Println("\n=== Testing Bidirectional Streaming RPC (Running Average) ===")

	ctx := context.Background()
	stream, err := client.RunningAverage(ctx)
	if err != nil {
		log.Printf("RunningAverage failed: %v", err)
		return
	}

	numbers := []float64{10, 20, 30, 40, 50}
	fmt.Printf("Sending numbers for running average: %v\n", numbers)

	// Start a goroutine to receive responses
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Printf("Failed to receive response: %v", err)
				return
			}
			fmt.Printf("Running average: %.2f (count: %d)\n", resp.Average, resp.Count)
		}
	}()

	// Send numbers
	for _, num := range numbers {
		if err := stream.Send(&calculatorv1.RunningAverageRequest{Number: num}); err != nil {
			log.Printf("Failed to send number: %v", err)
			return
		}
		time.Sleep(500 * time.Millisecond) // Delay to see the streaming effect
	}

	// Close the send direction
	if err := stream.CloseSend(); err != nil {
		log.Printf("Failed to close send: %v", err)
		return
	}

	// Give some time for the last response
	time.Sleep(200 * time.Millisecond)
}
