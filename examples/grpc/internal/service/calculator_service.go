//go:build examples

package service

import (
	"context"
	"io"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	calculatorv1 "github.com/jasoet/pkg/v2/examples/grpc/gen/calculator/v1"
)

// CalculatorService implements the CalculatorService gRPC service
type CalculatorService struct {
	calculatorv1.UnimplementedCalculatorServiceServer
}

// NewCalculatorService creates a new calculator service instance
func NewCalculatorService() *CalculatorService {
	return &CalculatorService{}
}

// Add implements basic addition
func (s *CalculatorService) Add(ctx context.Context, req *calculatorv1.AddRequest) (*calculatorv1.AddResponse, error) {
	log.Printf("Add called with a=%v, b=%v", req.A, req.B)
	result := req.A + req.B
	return &calculatorv1.AddResponse{Result: result}, nil
}

// Subtract implements basic subtraction
func (s *CalculatorService) Subtract(ctx context.Context, req *calculatorv1.SubtractRequest) (*calculatorv1.SubtractResponse, error) {
	log.Printf("Subtract called with a=%v, b=%v", req.A, req.B)
	result := req.A - req.B
	return &calculatorv1.SubtractResponse{Result: result}, nil
}

// Multiply implements basic multiplication
func (s *CalculatorService) Multiply(ctx context.Context, req *calculatorv1.MultiplyRequest) (*calculatorv1.MultiplyResponse, error) {
	log.Printf("Multiply called with a=%v, b=%v", req.A, req.B)
	result := req.A * req.B
	return &calculatorv1.MultiplyResponse{Result: result}, nil
}

// Divide implements basic division with error handling
func (s *CalculatorService) Divide(ctx context.Context, req *calculatorv1.DivideRequest) (*calculatorv1.DivideResponse, error) {
	log.Printf("Divide called with a=%v, b=%v", req.A, req.B)
	if req.B == 0 {
		return nil, status.Error(codes.InvalidArgument, "division by zero is not allowed")
	}
	result := req.A / req.B
	return &calculatorv1.DivideResponse{Result: result}, nil
}

// Factorial implements server streaming - sends factorial calculation steps
func (s *CalculatorService) Factorial(req *calculatorv1.FactorialRequest, stream calculatorv1.CalculatorService_FactorialServer) error {
	log.Printf("Factorial called with number=%v", req.Number)

	if req.Number < 0 {
		return status.Error(codes.InvalidArgument, "factorial is not defined for negative numbers")
	}

	if req.Number > 20 {
		return status.Error(codes.InvalidArgument, "factorial calculation limited to numbers <= 20 to prevent overflow")
	}

	factorial := int64(1)
	for i := int32(1); i <= req.Number; i++ {
		factorial *= int64(i)

		response := &calculatorv1.FactorialResponse{
			Step:   i,
			Result: factorial,
		}

		if err := stream.Send(response); err != nil {
			return err
		}
		log.Printf("Sent factorial step %d: %d", i, factorial)
	}

	return nil
}

// Sum implements client streaming - receives numbers and returns their sum
func (s *CalculatorService) Sum(stream calculatorv1.CalculatorService_SumServer) error {
	log.Printf("Sum called - waiting for numbers")

	var total float64
	count := 0

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			log.Printf("Sum completed: total=%v from %d numbers", total, count)
			return stream.SendAndClose(&calculatorv1.SumResponse{Total: total})
		}
		if err != nil {
			return err
		}

		total += req.Number
		count++
		log.Printf("Received number: %v (running total: %v)", req.Number, total)
	}
}

// RunningAverage implements bidirectional streaming - calculates running average
func (s *CalculatorService) RunningAverage(stream calculatorv1.CalculatorService_RunningAverageServer) error {
	log.Printf("RunningAverage called - starting bidirectional stream")

	var sum float64
	var count int32

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			log.Printf("RunningAverage stream ended")
			return nil
		}
		if err != nil {
			return err
		}

		sum += req.Number
		count++
		average := sum / float64(count)

		response := &calculatorv1.RunningAverageResponse{
			Average: average,
			Count:   count,
		}

		if err := stream.Send(response); err != nil {
			return err
		}

		log.Printf("Received %v, running average: %v (count: %d)", req.Number, average, count)
	}
}
