package concurrent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExecuteConcurrently(t *testing.T) {
	// Test case 1: All functions succeed
	t.Run("all functions succeed", func(t *testing.T) {
		funcs := map[string]Func[float64]{
			"func1": func(ctx context.Context) (float64, error) {
				return 1.0, nil
			},
			"func2": func(ctx context.Context) (float64, error) {
				return 2.0, nil
			},
			"func3": func(ctx context.Context) (float64, error) {
				return 3.0, nil
			},
		}

		results, err := ExecuteConcurrently[float64](context.Background(), funcs)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(results))
		assert.Equal(t, 1.0, results["func1"])
		assert.Equal(t, 2.0, results["func2"])
		assert.Equal(t, 3.0, results["func3"])
	})

	// Test case 2: One function fails
	t.Run("one function fails", func(t *testing.T) {
		expectedErr := errors.New("function failed")
		funcs := map[string]Func[float64]{
			"func1": func(ctx context.Context) (float64, error) {
				return 1.0, nil
			},
			"func2": func(ctx context.Context) (float64, error) {
				return 0.0, expectedErr
			},
			"func3": func(ctx context.Context) (float64, error) {
				return 3.0, nil
			},
		}

		results, err := ExecuteConcurrently[float64](context.Background(), funcs)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, results)
	})

	// Test case 3: Context cancellation
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel the context immediately

		funcs := map[string]Func[float64]{
			"func1": func(ctx context.Context) (float64, error) {
				if ctx.Err() != nil {
					return 0.0, ctx.Err()
				}
				return 1.0, nil
			},
		}

		results, err := ExecuteConcurrently[float64](ctx, funcs)
		assert.Error(t, err)
		assert.Nil(t, results)
	})

	// Test case 4: One function panics
	t.Run("one function panics", func(t *testing.T) {
		funcs := map[string]Func[float64]{
			"func1": func(ctx context.Context) (float64, error) {
				return 1.0, nil
			},
			"func2": func(ctx context.Context) (float64, error) {
				panic("something went wrong")
			},
			"func3": func(ctx context.Context) (float64, error) {
				return 3.0, nil
			},
		}

		results, err := ExecuteConcurrently[float64](context.Background(), funcs)
		assert.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "panic in")
		assert.Contains(t, err.Error(), "something went wrong")
	})

	// Test case 5: Causal error preferred over context.Canceled
	t.Run("causal error preferred over context canceled", func(t *testing.T) {
		causalErr := errors.New("real failure")
		funcs := map[string]Func[float64]{
			"failing": func(ctx context.Context) (float64, error) {
				return 0, causalErr
			},
			"waiting": func(ctx context.Context) (float64, error) {
				<-ctx.Done()
				return 0, ctx.Err()
			},
		}

		results, err := ExecuteConcurrently[float64](context.Background(), funcs)
		assert.Error(t, err)
		assert.Nil(t, results)
		// Should return the causal error, not context.Canceled
		assert.Equal(t, causalErr, err)
	})

	// Test case 6: Empty function map
	t.Run("empty function map", func(t *testing.T) {
		funcs := map[string]Func[float64]{}

		results, err := ExecuteConcurrently[float64](context.Background(), funcs)
		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	// Test case 7: Functions with delays
	t.Run("functions with delays", func(t *testing.T) {
		funcs := map[string]Func[float64]{
			"fast": func(ctx context.Context) (float64, error) {
				return 1.0, nil
			},
			"medium": func(ctx context.Context) (float64, error) {
				time.Sleep(50 * time.Millisecond)
				return 2.0, nil
			},
			"slow": func(ctx context.Context) (float64, error) {
				time.Sleep(100 * time.Millisecond)
				return 3.0, nil
			},
		}

		start := time.Now()
		results, err := ExecuteConcurrently[float64](context.Background(), funcs)
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.Equal(t, 3, len(results))
		assert.Equal(t, 1.0, results["fast"])
		assert.Equal(t, 2.0, results["medium"])
		assert.Equal(t, 3.0, results["slow"])

		// Verify that the execution time is closer to the slowest function than the sum of all functions
		assert.Less(t, elapsed, 150*time.Millisecond)           // Should be less than the sum (150ms)
		assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond) // Should be at least as long as the slowest function
	})
}

// ResultValue is an interface for different result types
type ResultValue interface {
	isResultValue()
}

// StringResult wraps a string value
type StringResult struct {
	Value string
}

func (StringResult) isResultValue() {}

// FloatResult wraps a float64 value
type FloatResult struct {
	Value float64
}

func (FloatResult) isResultValue() {}

// IntResult wraps an int value
type IntResult struct {
	Value int
}

func (IntResult) isResultValue() {}

func TestExecuteConcurrentlyWithInterface(t *testing.T) {
	// Define a test struct for mixed types
	type MixedTypeDTO struct {
		StringValue string
		FloatValue  float64
		IntValue    int
		Combined    string
	}

	// Test case: Using interface to handle different return types
	t.Run("using interface for different types", func(t *testing.T) {
		funcs := map[string]Func[ResultValue]{
			"stringValue": func(ctx context.Context) (ResultValue, error) {
				return StringResult{Value: "Hello"}, nil
			},
			"floatValue": func(ctx context.Context) (ResultValue, error) {
				return FloatResult{Value: 42.5}, nil
			},
			"intValue": func(ctx context.Context) (ResultValue, error) {
				return IntResult{Value: 100}, nil
			},
		}

		resultBuilder := func(results map[string]ResultValue) (MixedTypeDTO, error) {
			// Type assertions to extract the actual values
			stringVal := results["stringValue"].(StringResult).Value
			floatVal := results["floatValue"].(FloatResult).Value
			intVal := results["intValue"].(IntResult).Value

			return MixedTypeDTO{
				StringValue: stringVal,
				FloatValue:  floatVal,
				IntValue:    intVal,
				Combined:    stringVal + " " + string(rune(intVal)),
			}, nil
		}

		result, err := ExecuteConcurrentlyTyped[ResultValue, MixedTypeDTO](context.Background(), resultBuilder, funcs)
		assert.NoError(t, err)
		assert.Equal(t, "Hello", result.StringValue)
		assert.Equal(t, 42.5, result.FloatValue)
		assert.Equal(t, 100, result.IntValue)
		assert.Equal(t, "Hello d", result.Combined) // 'd' is ASCII 100
	})

	// Test case: Error handling with interface types
	t.Run("error handling with interface types", func(t *testing.T) {
		expectedErr := errors.New("function failed")
		funcs := map[string]Func[ResultValue]{
			"stringValue": func(ctx context.Context) (ResultValue, error) {
				return StringResult{Value: "Hello"}, nil
			},
			"floatValue": func(ctx context.Context) (ResultValue, error) {
				return nil, expectedErr
			},
		}

		resultBuilder := func(results map[string]ResultValue) (MixedTypeDTO, error) {
			return MixedTypeDTO{}, nil // Won't be called due to error
		}

		result, err := ExecuteConcurrentlyTyped[ResultValue, MixedTypeDTO](context.Background(), resultBuilder, funcs)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, MixedTypeDTO{}, result)
	})

	// Test case: Type switch for more robust type handling
	t.Run("type switch for robust type handling", func(t *testing.T) {
		funcs := map[string]Func[ResultValue]{
			"stringValue": func(ctx context.Context) (ResultValue, error) {
				return StringResult{Value: "Hello"}, nil
			},
			"floatValue": func(ctx context.Context) (ResultValue, error) {
				return FloatResult{Value: 42.5}, nil
			},
			"intValue": func(ctx context.Context) (ResultValue, error) {
				return IntResult{Value: 100}, nil
			},
		}

		resultBuilder := func(results map[string]ResultValue) (MixedTypeDTO, error) {
			var dto MixedTypeDTO

			// Using type switch for safer type handling
			for key, result := range results {
				switch v := result.(type) {
				case StringResult:
					if key == "stringValue" {
						dto.StringValue = v.Value
					}
				case FloatResult:
					if key == "floatValue" {
						dto.FloatValue = v.Value
					}
				case IntResult:
					if key == "intValue" {
						dto.IntValue = v.Value
					}
				default:
					return MixedTypeDTO{}, errors.New("unexpected result type")
				}
			}

			dto.Combined = dto.StringValue + " " + string(rune(dto.IntValue))
			return dto, nil
		}

		result, err := ExecuteConcurrentlyTyped[ResultValue, MixedTypeDTO](context.Background(), resultBuilder, funcs)
		assert.NoError(t, err)
		assert.Equal(t, "Hello", result.StringValue)
		assert.Equal(t, 42.5, result.FloatValue)
		assert.Equal(t, 100, result.IntValue)
		assert.Equal(t, "Hello d", result.Combined) // 'd' is ASCII 100
	})
}

func TestExecuteConcurrentlyTyped(t *testing.T) {
	// Define a test struct
	type TestDTO struct {
		Value1 float64
		Value2 float64
		Sum    float64
	}

	// Test case: Using a different type (string) to verify generic functionality
	t.Run("using string type", func(t *testing.T) {
		funcs := map[string]Func[string]{
			"greeting": func(ctx context.Context) (string, error) {
				return "Hello", nil
			},
			"name": func(ctx context.Context) (string, error) {
				return "World", nil
			},
		}

		type StringDTO struct {
			Greeting string
			Name     string
			Message  string
		}

		resultBuilder := func(results map[string]string) (StringDTO, error) {
			return StringDTO{
				Greeting: results["greeting"],
				Name:     results["name"],
				Message:  results["greeting"] + " " + results["name"],
			}, nil
		}

		result, err := ExecuteConcurrentlyTyped[string, StringDTO](context.Background(), resultBuilder, funcs)
		assert.NoError(t, err)
		assert.Equal(t, "Hello", result.Greeting)
		assert.Equal(t, "World", result.Name)
		assert.Equal(t, "Hello World", result.Message)
	})

	// Test case: Successful execution with result builder
	t.Run("successful execution with result builder", func(t *testing.T) {
		funcs := map[string]Func[float64]{
			"value1": func(ctx context.Context) (float64, error) {
				return 10.0, nil
			},
			"value2": func(ctx context.Context) (float64, error) {
				return 20.0, nil
			},
		}

		resultBuilder := func(results map[string]float64) (TestDTO, error) {
			return TestDTO{
				Value1: results["value1"],
				Value2: results["value2"],
				Sum:    results["value1"] + results["value2"],
			}, nil
		}

		result, err := ExecuteConcurrentlyTyped[float64, TestDTO](context.Background(), resultBuilder, funcs)
		assert.NoError(t, err)
		assert.Equal(t, 10.0, result.Value1)
		assert.Equal(t, 20.0, result.Value2)
		assert.Equal(t, 30.0, result.Sum)
	})

	// Test case: Error in repository function
	t.Run("error in repository function", func(t *testing.T) {
		expectedErr := errors.New("function failed")
		funcs := map[string]Func[float64]{
			"value1": func(ctx context.Context) (float64, error) {
				return 10.0, nil
			},
			"value2": func(ctx context.Context) (float64, error) {
				return 0.0, expectedErr
			},
		}

		resultBuilder := func(results map[string]float64) (TestDTO, error) {
			return TestDTO{
				Value1: results["value1"],
				Value2: results["value2"],
				Sum:    results["value1"] + results["value2"],
			}, nil
		}

		result, err := ExecuteConcurrentlyTyped[float64, TestDTO](context.Background(), resultBuilder, funcs)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, TestDTO{}, result)
	})

	// Test case: Error in result builder
	t.Run("error in result builder", func(t *testing.T) {
		funcs := map[string]Func[float64]{
			"value1": func(ctx context.Context) (float64, error) {
				return 10.0, nil
			},
			"value2": func(ctx context.Context) (float64, error) {
				return 20.0, nil
			},
		}

		expectedErr := errors.New("builder failed")
		resultBuilder := func(results map[string]float64) (TestDTO, error) {
			return TestDTO{}, expectedErr
		}

		result, err := ExecuteConcurrentlyTyped[float64, TestDTO](context.Background(), resultBuilder, funcs)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, TestDTO{}, result)
	})
}
