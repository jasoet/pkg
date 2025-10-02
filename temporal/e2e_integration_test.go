//go:build temporal

package temporal

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jasoet/pkg/v2/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// E2E Test workflows and activities
func OrderProcessingWorkflow(ctx workflow.Context, orderID string, customerID string, amount float64) (*OrderResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Order processing workflow started", "orderID", orderID, "customerID", customerID, "amount", amount)

	// Configure activity options with retries
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	result := &OrderResult{
		OrderID:    orderID,
		CustomerID: customerID,
		Amount:     amount,
		Status:     "processing",
		Steps:      make(map[string]string),
	}

	// Step 1: Validate Order
	logger.Info("Step 1: Validating order", "orderID", orderID)
	var validationResult string
	err := workflow.ExecuteActivity(ctx, ValidateOrderActivity, orderID, customerID, amount).Get(ctx, &validationResult)
	if err != nil {
		logger.Error("Order validation failed", "orderID", orderID, "error", err)
		result.Status = "validation_failed"
		result.Error = err.Error()
		return result, err
	}
	result.Steps["validation"] = validationResult

	// Step 2: Process Payment
	logger.Info("Step 2: Processing payment", "orderID", orderID, "amount", amount)
	var paymentID string
	err = workflow.ExecuteActivity(ctx, ProcessPaymentActivity, orderID, amount).Get(ctx, &paymentID)
	if err != nil {
		logger.Error("Payment processing failed", "orderID", orderID, "error", err)
		result.Status = "payment_failed"
		result.Error = err.Error()
		return result, err
	}
	result.PaymentID = paymentID
	result.Steps["payment"] = "completed"

	// Step 3: Reserve Inventory
	logger.Info("Step 3: Reserving inventory", "orderID", orderID)
	var reservationID string
	err = workflow.ExecuteActivity(ctx, ReserveInventoryActivity, orderID).Get(ctx, &reservationID)
	if err != nil {
		logger.Error("Inventory reservation failed", "orderID", orderID, "error", err)

		// Compensate: Refund payment
		logger.Info("Compensating: Refunding payment", "paymentID", paymentID)
		_ = workflow.ExecuteActivity(ctx, RefundPaymentActivity, paymentID).Get(ctx, nil)

		result.Status = "inventory_failed"
		result.Error = err.Error()
		return result, err
	}
	result.ReservationID = reservationID
	result.Steps["inventory"] = "reserved"

	// Step 4: Ship Order
	logger.Info("Step 4: Shipping order", "orderID", orderID)
	var trackingNumber string
	err = workflow.ExecuteActivity(ctx, ShipOrderActivity, orderID, customerID).Get(ctx, &trackingNumber)
	if err != nil {
		logger.Error("Shipping failed", "orderID", orderID, "error", err)

		// Compensate: Release inventory and refund payment
		logger.Info("Compensating: Releasing inventory", "reservationID", reservationID)
		_ = workflow.ExecuteActivity(ctx, ReleaseInventoryActivity, reservationID).Get(ctx, nil)

		logger.Info("Compensating: Refunding payment", "paymentID", paymentID)
		_ = workflow.ExecuteActivity(ctx, RefundPaymentActivity, paymentID).Get(ctx, nil)

		result.Status = "shipping_failed"
		result.Error = err.Error()
		return result, err
	}
	result.TrackingNumber = trackingNumber
	result.Steps["shipping"] = "shipped"

	// Step 5: Send Confirmation
	logger.Info("Step 5: Sending confirmation", "orderID", orderID)
	err = workflow.ExecuteActivity(ctx, SendConfirmationActivity, orderID, customerID, trackingNumber).Get(ctx, nil)
	if err != nil {
		// Non-critical step, log but don't fail the workflow
		logger.Warn("Confirmation sending failed (non-critical)", "orderID", orderID, "error", err)
	} else {
		result.Steps["confirmation"] = "sent"
	}

	result.Status = "completed"
	result.CompletedAt = workflow.Now(ctx)

	logger.Info("Order processing workflow completed successfully",
		"orderID", orderID,
		"paymentID", paymentID,
		"trackingNumber", trackingNumber)

	return result, nil
}

// Activities for E2E testing
func ValidateOrderActivity(ctx context.Context, orderID string, customerID string, amount float64) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Validating order", "orderID", orderID, "customerID", customerID, "amount", amount)

	// Simulate validation logic
	if amount <= 0 {
		return "", fmt.Errorf("invalid order amount: %f", amount)
	}
	if customerID == "" {
		return "", fmt.Errorf("missing customer ID")
	}
	if orderID == "" {
		return "", fmt.Errorf("missing order ID")
	}

	// Simulate network delay
	time.Sleep(200 * time.Millisecond)

	result := fmt.Sprintf("Order %s validated for customer %s", orderID, customerID)
	logger.Info("Order validation completed", "orderID", orderID, "result", result)
	return result, nil
}

func ProcessPaymentActivity(ctx context.Context, orderID string, amount float64) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Processing payment", "orderID", orderID, "amount", amount)

	// Simulate payment processing delay
	time.Sleep(500 * time.Millisecond)

	// Simulate occasional payment failure only for specific test orders
	if strings.Contains(orderID, "invalid_order") {
		return "", fmt.Errorf("payment declined for order %s", orderID)
	}

	paymentID := fmt.Sprintf("pay_%s_%d", orderID, time.Now().Unix())
	logger.Info("Payment processed successfully", "orderID", orderID, "paymentID", paymentID)
	return paymentID, nil
}

func ReserveInventoryActivity(ctx context.Context, orderID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Reserving inventory", "orderID", orderID)

	// Simulate inventory check delay
	time.Sleep(300 * time.Millisecond)

	// Simulate inventory shortage only for specific test orders
	if strings.Contains(orderID, "inventory_fail") {
		return "", fmt.Errorf("insufficient inventory for order %s", orderID)
	}

	reservationID := fmt.Sprintf("res_%s_%d", orderID, time.Now().Unix())
	logger.Info("Inventory reserved successfully", "orderID", orderID, "reservationID", reservationID)
	return reservationID, nil
}

func ShipOrderActivity(ctx context.Context, orderID string, customerID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Shipping order", "orderID", orderID, "customerID", customerID)

	// Simulate shipping arrangement delay
	time.Sleep(400 * time.Millisecond)

	// Simulate shipping failure only for specific test orders
	if strings.Contains(orderID, "shipping_fail") {
		return "", fmt.Errorf("shipping carrier unavailable for order %s", orderID)
	}

	trackingNumber := fmt.Sprintf("TRK%d", time.Now().UnixNano())
	logger.Info("Order shipped successfully", "orderID", orderID, "trackingNumber", trackingNumber)
	return trackingNumber, nil
}

func SendConfirmationActivity(ctx context.Context, orderID string, customerID string, trackingNumber string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending confirmation", "orderID", orderID, "customerID", customerID, "trackingNumber", trackingNumber)

	// Simulate email sending delay
	time.Sleep(100 * time.Millisecond)

	logger.Info("Confirmation sent successfully", "orderID", orderID, "trackingNumber", trackingNumber)
	return nil
}

// Compensation activities
func RefundPaymentActivity(ctx context.Context, paymentID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Refunding payment", "paymentID", paymentID)

	// Simulate refund processing delay
	time.Sleep(300 * time.Millisecond)

	logger.Info("Payment refunded successfully", "paymentID", paymentID)
	return nil
}

func ReleaseInventoryActivity(ctx context.Context, reservationID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Releasing inventory reservation", "reservationID", reservationID)

	// Simulate inventory release delay
	time.Sleep(200 * time.Millisecond)

	logger.Info("Inventory reservation released successfully", "reservationID", reservationID)
	return nil
}

// Result types
type OrderResult struct {
	OrderID        string            `json:"orderId"`
	CustomerID     string            `json:"customerId"`
	Amount         float64           `json:"amount"`
	Status         string            `json:"status"`
	PaymentID      string            `json:"paymentId,omitempty"`
	ReservationID  string            `json:"reservationId,omitempty"`
	TrackingNumber string            `json:"trackingNumber,omitempty"`
	Steps          map[string]string `json:"steps"`
	Error          string            `json:"error,omitempty"`
	CompletedAt    time.Time         `json:"completedAt"`
}

// E2E Integration Tests
func TestE2EOrderProcessingWorkflow(t *testing.T) {
	// Initialize logging for integration tests
	logging.Initialize("temporal-integration-test", true)

	ctx := context.Background()

	// Start Temporal container once for all subtests
	container, _, containerCleanup := setupTemporalContainerForTest(ctx, t)
	defer containerCleanup()

	// Create config using container's address
	config := DefaultConfig()
	config.HostPort = container.HostPort
	config.MetricsListenAddress = "0.0.0.0:0" // Random port

	wm, err := NewWorkerManager(config)
	require.NoError(t, err, "Failed to create WorkerManager")
	defer wm.Close()

	taskQueue := "e2e-order-processing"

	t.Run("SuccessfulOrderProcessing", func(t *testing.T) {
		// Register worker with all workflows and activities
		w := wm.Register(taskQueue, worker.Options{})
		w.RegisterWorkflow(OrderProcessingWorkflow)
		w.RegisterActivity(ValidateOrderActivity)
		w.RegisterActivity(ProcessPaymentActivity)
		w.RegisterActivity(ReserveInventoryActivity)
		w.RegisterActivity(ShipOrderActivity)
		w.RegisterActivity(SendConfirmationActivity)
		w.RegisterActivity(RefundPaymentActivity)
		w.RegisterActivity(ReleaseInventoryActivity)

		// Start worker
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = wm.Start(ctx, w)
		}()

		// Give worker time to start
		time.Sleep(3 * time.Second)

		// Execute workflow
		temporalClient := wm.GetClient()
		orderID := fmt.Sprintf("order_%d", time.Now().Unix())
		customerID := "customer_123"
		amount := 99.99

		options := client.StartWorkflowOptions{
			ID:        fmt.Sprintf("e2e-order-processing-%s", orderID),
			TaskQueue: taskQueue,
		}

		workflowCtx, workflowCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer workflowCancel()

		workflowRun, err := temporalClient.ExecuteWorkflow(workflowCtx, options, OrderProcessingWorkflow, orderID, customerID, amount)
		require.NoError(t, err, "Failed to start order processing workflow")

		// Get result
		var result OrderResult
		err = workflowRun.Get(workflowCtx, &result)
		require.NoError(t, err, "Failed to get workflow result")

		// Verify successful completion
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, orderID, result.OrderID)
		assert.Equal(t, customerID, result.CustomerID)
		assert.Equal(t, amount, result.Amount)
		assert.NotEmpty(t, result.PaymentID)
		assert.NotEmpty(t, result.ReservationID)
		assert.NotEmpty(t, result.TrackingNumber)
		assert.Contains(t, result.Steps, "validation")
		assert.Contains(t, result.Steps, "payment")
		assert.Contains(t, result.Steps, "inventory")
		assert.Contains(t, result.Steps, "shipping")
		assert.Empty(t, result.Error)
		assert.False(t, result.CompletedAt.IsZero())

		w.Stop()
	})

	t.Run("OrderValidationFailure", func(t *testing.T) {
		// Register worker
		w := wm.Register(taskQueue+"-validation", worker.Options{})
		w.RegisterWorkflow(OrderProcessingWorkflow)
		w.RegisterActivity(ValidateOrderActivity)
		w.RegisterActivity(ProcessPaymentActivity)
		w.RegisterActivity(ReserveInventoryActivity)
		w.RegisterActivity(ShipOrderActivity)
		w.RegisterActivity(SendConfirmationActivity)
		w.RegisterActivity(RefundPaymentActivity)
		w.RegisterActivity(ReleaseInventoryActivity)

		// Start worker
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = wm.Start(ctx, w)
		}()

		time.Sleep(2 * time.Second)

		// Execute workflow with invalid data (negative amount)
		temporalClient := wm.GetClient()
		orderID := fmt.Sprintf("invalid_order_%d", time.Now().Unix())
		customerID := "customer_456"
		amount := -50.0 // Invalid amount

		options := client.StartWorkflowOptions{
			ID:        fmt.Sprintf("e2e-validation-fail-%s", orderID),
			TaskQueue: taskQueue + "-validation",
		}

		workflowCtx, workflowCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer workflowCancel()

		workflowRun, err := temporalClient.ExecuteWorkflow(workflowCtx, options, OrderProcessingWorkflow, orderID, customerID, amount)
		require.NoError(t, err, "Failed to start workflow")

		// Expect workflow to fail at validation
		var result OrderResult
		err = workflowRun.Get(workflowCtx, &result)
		assert.Error(t, err, "Workflow should have failed at validation")

		w.Stop()
	})

	t.Run("MultipleOrdersParallel", func(t *testing.T) {
		// Test processing multiple orders in parallel
		w := wm.Register(taskQueue+"-parallel", worker.Options{
			MaxConcurrentWorkflowTaskExecutionSize: 10,
		})
		w.RegisterWorkflow(OrderProcessingWorkflow)
		w.RegisterActivity(ValidateOrderActivity)
		w.RegisterActivity(ProcessPaymentActivity)
		w.RegisterActivity(ReserveInventoryActivity)
		w.RegisterActivity(ShipOrderActivity)
		w.RegisterActivity(SendConfirmationActivity)
		w.RegisterActivity(RefundPaymentActivity)
		w.RegisterActivity(ReleaseInventoryActivity)

		// Start worker
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = wm.Start(ctx, w)
		}()

		time.Sleep(2 * time.Second)

		temporalClient := wm.GetClient()
		orderCount := 5
		workflows := make([]client.WorkflowRun, orderCount)

		// Start multiple workflows
		for i := 0; i < orderCount; i++ {
			orderID := fmt.Sprintf("parallel_order_%d_%d", time.Now().Unix(), i)
			customerID := fmt.Sprintf("customer_%d", i)
			amount := float64(100 + i*10)

			options := client.StartWorkflowOptions{
				ID:        fmt.Sprintf("e2e-parallel-%s", orderID),
				TaskQueue: taskQueue + "-parallel",
			}

			workflowCtx, _ := context.WithTimeout(context.Background(), 60*time.Second)
			workflowRun, err := temporalClient.ExecuteWorkflow(workflowCtx, options, OrderProcessingWorkflow, orderID, customerID, amount)
			require.NoError(t, err, "Failed to start parallel workflow %d", i)
			workflows[i] = workflowRun
		}

		// Wait for all workflows to complete
		completedCount := 0
		for i, workflowRun := range workflows {
			workflowCtx, workflowCancel := context.WithTimeout(context.Background(), 60*time.Second)
			var result OrderResult
			err := workflowRun.Get(workflowCtx, &result)
			workflowCancel()

			if err == nil && result.Status == "completed" {
				completedCount++
				t.Logf("Parallel workflow %d completed successfully", i)
			} else {
				t.Logf("Parallel workflow %d failed: %v", i, err)
			}
		}

		w.Stop()
	})
}

func TestE2ETemporalIntegration(t *testing.T) {
	logging.Initialize("temporal-full-integration", true)

	ctx := context.Background()

	// Start Temporal container once for all subtests
	container, _, containerCleanup := setupTemporalContainerForTest(ctx, t)
	defer containerCleanup()

	// Create config using container's address
	config := DefaultConfig()
	config.HostPort = container.HostPort
	config.MetricsListenAddress = "0.0.0.0:0" // Random port

	t.Run("FullTemporalStackTest", func(t *testing.T) {
		// Test the full Temporal stack integration

		// 1. Create client
		temporalClient, err := NewClient(config)
		require.NoError(t, err, "Failed to create Temporal client")
		defer temporalClient.Close()

		// 2. Create worker manager
		wm, err := NewWorkerManager(config)
		require.NoError(t, err, "Failed to create WorkerManager")
		defer wm.Close()

		// 3. Create schedule manager
		sm := NewScheduleManager(temporalClient)
		require.NotNil(t, sm, "ScheduleManager should not be nil")

		// 4. Test basic connectivity
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test connectivity by checking workflow service
		workflowService := temporalClient.WorkflowService()
		require.NotNil(t, workflowService, "Failed to get workflow service")

		// 5. Register a simple worker
		taskQueue := "full-integration-test"
		w := wm.Register(taskQueue, worker.Options{})

		simpleWorkflow := func(ctx workflow.Context, message string) (string, error) {
			return fmt.Sprintf("Hello %s from Temporal!", message), nil
		}

		w.RegisterWorkflow(simpleWorkflow)

		// Start worker
		go func() {
			_ = wm.Start(ctx, w)
		}()

		time.Sleep(2 * time.Second)

		// 6. Execute simple workflow
		options := client.StartWorkflowOptions{
			ID:        fmt.Sprintf("full-integration-%d", time.Now().Unix()),
			TaskQueue: taskQueue,
		}

		workflowRun, err := temporalClient.ExecuteWorkflow(ctx, options, simpleWorkflow, "Integration Test")
		require.NoError(t, err, "Failed to execute simple workflow")

		var result string
		err = workflowRun.Get(ctx, &result)
		require.NoError(t, err, "Failed to get workflow result")

		assert.Equal(t, "Hello Integration Test from Temporal!", result)

		// 7. Test workflow queries and signals (if applicable)
		workflowInfo := workflowRun.GetID()
		assert.NotEmpty(t, workflowInfo, "Workflow ID should not be empty")

		t.Logf("Full Temporal integration test completed successfully. Workflow ID: %s", workflowInfo)
	})
}
