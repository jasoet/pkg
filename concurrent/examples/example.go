//go:build example

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jasoet/pkg/concurrent"
)

// Example data structures
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type Order struct {
	ID     int     `json:"id"`
	UserID int     `json:"user_id"`
	Total  float64 `json:"total"`
	Status string  `json:"status"`
}

type ApiResponse struct {
	Data      interface{} `json:"data"`
	Status    string      `json:"status"`
	Timestamp time.Time   `json:"timestamp"`
}

type DashboardData struct {
	Users    []User    `json:"users"`
	Products []Product `json:"products"`
	Orders   []Order   `json:"orders"`
}

type ProcessedItem struct {
	ID     int    `json:"id"`
	Result string `json:"result"`
}

func main() {
	fmt.Println("Concurrent Package Examples")
	fmt.Println("===========================")

	// Example 1: Basic Concurrent Execution
	fmt.Println("\n1. Basic Concurrent Execution")
	basicConcurrentExample()

	// Example 2: Database Operations Concurrency
	fmt.Println("\n2. Database Operations Concurrency")
	databaseConcurrencyExample()

	// Example 3: Typed Result Building
	fmt.Println("\n3. Typed Result Building")
	typedResultExample()

	// Example 4: API Calls Concurrency
	fmt.Println("\n4. API Calls Concurrency")
	apiConcurrencyExample()

	// Example 5: Error Handling and Context Cancellation
	fmt.Println("\n5. Error Handling and Context Cancellation")
	errorHandlingExample()

	// Example 6: Batch Processing
	fmt.Println("\n6. Batch Processing")
	batchProcessingExample()

	// Example 7: Performance Comparison
	fmt.Println("\n7. Performance Comparison")
	performanceComparisonExample()
}

func basicConcurrentExample() {
	ctx := context.Background()

	// Define functions that simulate different tasks
	funcs := map[string]concurrent.Func[string]{
		"task1": func(ctx context.Context) (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "Result from task 1", nil
		},
		"task2": func(ctx context.Context) (string, error) {
			time.Sleep(200 * time.Millisecond)
			return "Result from task 2", nil
		},
		"task3": func(ctx context.Context) (string, error) {
			time.Sleep(150 * time.Millisecond)
			return "Result from task 3", nil
		},
	}

	start := time.Now()
	results, err := concurrent.ExecuteConcurrently(ctx, funcs)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("Error in concurrent execution: %v", err)
		return
	}

	fmt.Printf("Concurrent execution completed in %v\n", elapsed)
	for key, result := range results {
		fmt.Printf("- %s: %s\n", key, result)
	}

	// Sequential comparison
	start = time.Now()
	for key, fn := range funcs {
		result, err := fn(ctx)
		if err != nil {
			log.Printf("Error in %s: %v", key, err)
			continue
		}
		fmt.Printf("Sequential %s: %s\n", key, result)
	}
	sequentialElapsed := time.Since(start)

	fmt.Printf("Sequential execution took %v (%.2fx slower)\n",
		sequentialElapsed, float64(sequentialElapsed)/float64(elapsed))
}

func databaseConcurrencyExample() {
	ctx := context.Background()

	// Simulate database operations
	dbFuncs := map[string]concurrent.Func[[]User]{
		"active_users": func(ctx context.Context) ([]User, error) {
			time.Sleep(100 * time.Millisecond) // Simulate DB query
			return []User{
				{ID: 1, Name: "Alice", Role: "user"},
				{ID: 2, Name: "Bob", Role: "user"},
			}, nil
		},
		"inactive_users": func(ctx context.Context) ([]User, error) {
			time.Sleep(150 * time.Millisecond) // Simulate DB query
			return []User{
				{ID: 3, Name: "Charlie", Role: "user"},
			}, nil
		},
		"admin_users": func(ctx context.Context) ([]User, error) {
			time.Sleep(80 * time.Millisecond) // Simulate DB query
			return []User{
				{ID: 4, Name: "Admin", Role: "admin"},
			}, nil
		},
	}

	start := time.Now()
	userGroups, err := concurrent.ExecuteConcurrently(ctx, dbFuncs)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("Error in database operations: %v", err)
		return
	}

	fmt.Printf("Database operations completed in %v\n", elapsed)
	for groupName, users := range userGroups {
		fmt.Printf("- %s: %d users\n", groupName, len(users))
		for _, user := range users {
			fmt.Printf("  * %s (ID: %d, Role: %s)\n", user.Name, user.ID, user.Role)
		}
	}
}

func typedResultExample() {
	ctx := context.Background()

	// Define functions that return different types but we'll use interface{}
	dataFuncs := map[string]concurrent.Func[interface{}]{
		"users": func(ctx context.Context) (interface{}, error) {
			time.Sleep(100 * time.Millisecond)
			return []User{
				{ID: 1, Name: "Alice", Role: "user"},
				{ID: 2, Name: "Bob", Role: "admin"},
			}, nil
		},
		"products": func(ctx context.Context) (interface{}, error) {
			time.Sleep(120 * time.Millisecond)
			return []Product{
				{ID: 1, Name: "Laptop", Price: 999.99},
				{ID: 2, Name: "Mouse", Price: 29.99},
			}, nil
		},
		"orders": func(ctx context.Context) (interface{}, error) {
			time.Sleep(80 * time.Millisecond)
			return []Order{
				{ID: 1, UserID: 1, Total: 1029.98, Status: "completed"},
				{ID: 2, UserID: 2, Total: 29.99, Status: "pending"},
			}, nil
		},
	}

	// Result builder function
	resultBuilder := func(results map[string]interface{}) (DashboardData, error) {
		dashboard := DashboardData{}

		if users, ok := results["users"].([]User); ok {
			dashboard.Users = users
		} else {
			return dashboard, fmt.Errorf("invalid users data type")
		}

		if products, ok := results["products"].([]Product); ok {
			dashboard.Products = products
		} else {
			return dashboard, fmt.Errorf("invalid products data type")
		}

		if orders, ok := results["orders"].([]Order); ok {
			dashboard.Orders = orders
		} else {
			return dashboard, fmt.Errorf("invalid orders data type")
		}

		return dashboard, nil
	}

	start := time.Now()
	dashboard, err := concurrent.ExecuteConcurrentlyTyped(ctx, resultBuilder, dataFuncs)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("Error in typed result building: %v", err)
		return
	}

	fmt.Printf("Dashboard data loaded in %v\n", elapsed)
	fmt.Printf("- Users: %d\n", len(dashboard.Users))
	fmt.Printf("- Products: %d\n", len(dashboard.Products))
	fmt.Printf("- Orders: %d\n", len(dashboard.Orders))

	// Display summary
	totalRevenue := 0.0
	for _, order := range dashboard.Orders {
		if order.Status == "completed" {
			totalRevenue += order.Total
		}
	}
	fmt.Printf("- Total Revenue: $%.2f\n", totalRevenue)
}

func apiConcurrencyExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Simulate API calls
	apiFuncs := map[string]concurrent.Func[ApiResponse]{
		"weather": func(ctx context.Context) (ApiResponse, error) {
			time.Sleep(200 * time.Millisecond) // Simulate API call
			return ApiResponse{
				Data:      map[string]interface{}{"temperature": 72, "humidity": 65},
				Status:    "success",
				Timestamp: time.Now(),
			}, nil
		},
		"news": func(ctx context.Context) (ApiResponse, error) {
			time.Sleep(300 * time.Millisecond) // Simulate API call
			return ApiResponse{
				Data:      []string{"Tech news 1", "Tech news 2", "Tech news 3"},
				Status:    "success",
				Timestamp: time.Now(),
			}, nil
		},
		"stock": func(ctx context.Context) (ApiResponse, error) {
			time.Sleep(150 * time.Millisecond) // Simulate API call
			return ApiResponse{
				Data:      map[string]float64{"AAPL": 150.25, "GOOGL": 2750.80},
				Status:    "success",
				Timestamp: time.Now(),
			}, nil
		},
	}

	start := time.Now()
	apiResults, err := concurrent.ExecuteConcurrently(ctx, apiFuncs)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("Error in API calls: %v", err)
		return
	}

	fmt.Printf("API calls completed in %v\n", elapsed)
	for apiName, response := range apiResults {
		fmt.Printf("- %s API: %s (called at %v)\n",
			apiName, response.Status, response.Timestamp.Format("15:04:05.000"))
	}
}

func errorHandlingExample() {
	ctx := context.Background()

	// Functions with different behaviors
	funcs := map[string]concurrent.Func[string]{
		"success": func(ctx context.Context) (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "Success", nil
		},
		"failure": func(ctx context.Context) (string, error) {
			time.Sleep(50 * time.Millisecond)
			return "", errors.New("operation failed")
		},
		"slow": func(ctx context.Context) (string, error) {
			select {
			case <-time.After(1 * time.Second):
				return "Slow result", nil
			case <-ctx.Done():
				return "", fmt.Errorf("operation cancelled: %w", ctx.Err())
			}
		},
	}

	fmt.Println("Executing functions with error (fail-fast behavior):")
	start := time.Now()
	results, err := concurrent.ExecuteConcurrently(ctx, funcs)
	elapsed := time.Since(start)

	fmt.Printf("Execution completed in %v\n", elapsed)
	if err != nil {
		fmt.Printf("Error encountered (as expected): %v\n", err)
		fmt.Println("Notice: The slow operation was cancelled due to the failure")
	} else {
		fmt.Println("All operations succeeded:")
		for key, result := range results {
			fmt.Printf("- %s: %s\n", key, result)
		}
	}

	// Example of handling errors gracefully
	fmt.Println("\nExample of graceful error handling:")
	gracefulErrorHandling()
}

func gracefulErrorHandling() {
	ctx := context.Background()

	// Retry mechanism
	maxRetries := 2
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("Attempt %d:\n", attempt)

		funcs := map[string]concurrent.Func[string]{
			"task1": func(ctx context.Context) (string, error) {
				// Simulate random failure
				if rand.Float32() < 0.3 { // 30% chance of failure
					return "", errors.New("random failure")
				}
				return "Task 1 completed", nil
			},
			"task2": func(ctx context.Context) (string, error) {
				time.Sleep(100 * time.Millisecond)
				return "Task 2 completed", nil
			},
		}

		results, err := concurrent.ExecuteConcurrently(ctx, funcs)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			if attempt < maxRetries {
				fmt.Println("  Retrying...")
				time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
				continue
			}
			fmt.Println("  Max retries reached, giving up")
			return
		}

		fmt.Println("  Success!")
		for key, result := range results {
			fmt.Printf("  - %s: %s\n", key, result)
		}
		return
	}
}

func batchProcessingExample() {
	ctx := context.Background()

	// Simulate a large dataset
	items := make([]int, 50)
	for i := range items {
		items[i] = i + 1
	}

	fmt.Printf("Processing %d items in batches...\n", len(items))

	results, err := processBatch(ctx, items)
	if err != nil {
		log.Printf("Error in batch processing: %v", err)
		return
	}

	fmt.Printf("Processed %d items successfully\n", len(results))

	// Show first few results
	for i, result := range results {
		if i >= 5 {
			fmt.Printf("... and %d more\n", len(results)-5)
			break
		}
		fmt.Printf("- Item %d: %s\n", result.ID, result.Result)
	}
}

func processBatch(ctx context.Context, items []int) ([]ProcessedItem, error) {
	batchSize := 10
	batches := make(map[string]concurrent.Func[[]ProcessedItem])

	// Create batches
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		batchKey := fmt.Sprintf("batch_%d", i/batchSize)

		batches[batchKey] = func(batch []int) concurrent.Func[[]ProcessedItem] {
			return func(ctx context.Context) ([]ProcessedItem, error) {
				return processItems(ctx, batch)
			}
		}(batch) // Capture batch variable
	}

	start := time.Now()
	batchResults, err := concurrent.ExecuteConcurrently(ctx, batches)
	elapsed := time.Since(start)

	if err != nil {
		return nil, err
	}

	// Combine all results
	var allResults []ProcessedItem
	for batchName, batchResult := range batchResults {
		fmt.Printf("- %s: %d items\n", batchName, len(batchResult))
		allResults = append(allResults, batchResult...)
	}

	fmt.Printf("Batch processing completed in %v\n", elapsed)
	return allResults, nil
}

func processItems(ctx context.Context, items []int) ([]ProcessedItem, error) {
	// Simulate item processing
	time.Sleep(time.Duration(len(items)*10) * time.Millisecond)

	results := make([]ProcessedItem, len(items))
	for i, item := range items {
		results[i] = ProcessedItem{
			ID:     item,
			Result: fmt.Sprintf("Processed item %d", item),
		}
	}

	return results, nil
}

func performanceComparisonExample() {
	ctx := context.Background()

	// Create a set of CPU-bound and I/O-bound operations
	operations := map[string]concurrent.Func[string]{
		"cpu_task_1": cpuBoundTask("CPU Task 1", 100),
		"cpu_task_2": cpuBoundTask("CPU Task 2", 150),
		"io_task_1":  ioBoundTask("I/O Task 1", 200*time.Millisecond),
		"io_task_2":  ioBoundTask("I/O Task 2", 100*time.Millisecond),
		"io_task_3":  ioBoundTask("I/O Task 3", 300*time.Millisecond),
	}

	// Concurrent execution
	fmt.Println("Running concurrent execution...")
	start := time.Now()
	results, err := concurrent.ExecuteConcurrently(ctx, operations)
	concurrentTime := time.Since(start)

	if err != nil {
		log.Printf("Error in concurrent execution: %v", err)
		return
	}

	fmt.Printf("Concurrent execution: %v\n", concurrentTime)

	// Sequential execution for comparison
	fmt.Println("Running sequential execution...")
	start = time.Now()
	for name, op := range operations {
		result, err := op(ctx)
		if err != nil {
			log.Printf("Error in %s: %v", name, err)
			continue
		}
		_ = result // Use result
	}
	sequentialTime := time.Since(start)

	fmt.Printf("Sequential execution: %v\n", sequentialTime)
	fmt.Printf("Speedup: %.2fx faster\n", float64(sequentialTime)/float64(concurrentTime))

	// Show results
	fmt.Println("Results:")
	for name, result := range results {
		fmt.Printf("- %s: %s\n", name, result)
	}
}

func cpuBoundTask(name string, iterations int) concurrent.Func[string] {
	return func(ctx context.Context) (string, error) {
		// Simulate CPU-bound work
		sum := 0
		for i := 0; i < iterations*1000; i++ {
			sum += i
		}
		return fmt.Sprintf("%s completed (sum: %d)", name, sum), nil
	}
}

func ioBoundTask(name string, duration time.Duration) concurrent.Func[string] {
	return func(ctx context.Context) (string, error) {
		// Simulate I/O-bound work
		select {
		case <-time.After(duration):
			return fmt.Sprintf("%s completed after %v", name, duration), nil
		case <-ctx.Done():
			return "", fmt.Errorf("%s cancelled: %w", name, ctx.Err())
		}
	}
}

// Helper function to simulate external service calls
func simulateExternalCall(service string, delay time.Duration) concurrent.Func[string] {
	return func(ctx context.Context) (string, error) {
		select {
		case <-time.After(delay):
			return fmt.Sprintf("Response from %s", service), nil
		case <-ctx.Done():
			return "", fmt.Errorf("call to %s cancelled: %w", service, ctx.Err())
		}
	}
}

// Helper function to demonstrate different result types
func createTypedFunctions() map[string]concurrent.Func[interface{}] {
	return map[string]concurrent.Func[interface{}]{
		"string_result": func(ctx context.Context) (interface{}, error) {
			return "Hello, World!", nil
		},
		"int_result": func(ctx context.Context) (interface{}, error) {
			return 42, nil
		},
		"slice_result": func(ctx context.Context) (interface{}, error) {
			return []string{"apple", "banana", "cherry"}, nil
		},
		"map_result": func(ctx context.Context) (interface{}, error) {
			return map[string]int{"a": 1, "b": 2, "c": 3}, nil
		},
	}
}
