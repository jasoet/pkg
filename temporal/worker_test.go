package temporal

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// fakeWorker is a minimal worker.Worker used to exercise WorkerManager lifecycle
// logic without a real Temporal connection.
type fakeWorker struct {
	startErr   error
	startCalls int32
	stopCalls  int32
}

func (f *fakeWorker) RegisterWorkflow(any)                                         {}
func (f *fakeWorker) RegisterWorkflowWithOptions(any, workflow.RegisterOptions)    {}
func (f *fakeWorker) RegisterDynamicWorkflow(any, workflow.DynamicRegisterOptions) {}
func (f *fakeWorker) RegisterActivity(any)                                         {}
func (f *fakeWorker) RegisterActivityWithOptions(any, activity.RegisterOptions)    {}
func (f *fakeWorker) RegisterDynamicActivity(any, activity.DynamicRegisterOptions) {}
func (f *fakeWorker) RegisterNexusService(*nexus.Service)                          {}
func (f *fakeWorker) Start() error {
	atomic.AddInt32(&f.startCalls, 1)
	return f.startErr
}
func (f *fakeWorker) Run(<-chan interface{}) error { return nil }
func (f *fakeWorker) Stop()                        { atomic.AddInt32(&f.stopCalls, 1) }

func newManagerWithWorkers(t *testing.T, workers ...worker.Worker) *WorkerManager {
	t.Helper()
	wm, err := NewWorkerManager(mocks.NewClient(t))
	require.NoError(t, err)
	wm.workers = append(wm.workers, workers...)
	return wm
}

// TestStartAll_RollsBackOnFailure verifies that when a worker fails to start,
// the already-started workers are stopped before StartAll returns the error, so
// none keep polling against a half-initialized application.
func TestStartAll_RollsBackOnFailure(t *testing.T) {
	w0 := &fakeWorker{}
	w1 := &fakeWorker{}
	w2 := &fakeWorker{startErr: errors.New("boom")}
	wm := newManagerWithWorkers(t, w0, w1, w2)

	err := wm.StartAll(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")

	assert.Equal(t, int32(1), atomic.LoadInt32(&w0.startCalls))
	assert.Equal(t, int32(1), atomic.LoadInt32(&w1.startCalls))
	assert.Equal(t, int32(1), atomic.LoadInt32(&w2.startCalls))

	// The two successfully-started workers must be rolled back.
	assert.Equal(t, int32(1), atomic.LoadInt32(&w0.stopCalls), "started worker must be stopped on rollback")
	assert.Equal(t, int32(1), atomic.LoadInt32(&w1.stopCalls), "started worker must be stopped on rollback")
	// The worker that failed to start is not stopped.
	assert.Equal(t, int32(0), atomic.LoadInt32(&w2.stopCalls), "failed worker must not be stopped")
}

// TestStartAll_SuccessStartsAllWithoutStopping verifies the happy path leaves
// every worker running.
func TestStartAll_SuccessStartsAllWithoutStopping(t *testing.T) {
	w0 := &fakeWorker{}
	w1 := &fakeWorker{}
	wm := newManagerWithWorkers(t, w0, w1)

	require.NoError(t, wm.StartAll(context.Background()))
	assert.Equal(t, int32(1), atomic.LoadInt32(&w0.startCalls))
	assert.Equal(t, int32(1), atomic.LoadInt32(&w1.startCalls))
	assert.Zero(t, atomic.LoadInt32(&w0.stopCalls))
	assert.Zero(t, atomic.LoadInt32(&w1.stopCalls))
}

// TestClose_Idempotent verifies Close stops each worker at most once even when
// called multiple times, avoiding the SDK's double-Stop panic.
func TestClose_Idempotent(t *testing.T) {
	w0 := &fakeWorker{}
	w1 := &fakeWorker{}
	wm := newManagerWithWorkers(t, w0, w1)

	ctx := context.Background()
	wm.Close(ctx)
	wm.Close(ctx)
	wm.Close(ctx)

	assert.Equal(t, int32(1), atomic.LoadInt32(&w0.stopCalls), "worker must be stopped exactly once")
	assert.Equal(t, int32(1), atomic.LoadInt32(&w1.stopCalls), "worker must be stopped exactly once")
}
