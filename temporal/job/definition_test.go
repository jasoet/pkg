package job

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func TestNew_RequiresName(t *testing.T) {
	_, err := New("", "tq",
		WithRegister(func(worker.Worker) {}),
		WithExecute(func(context.Context, client.Client, client.StartWorkflowOptions, any) (client.WorkflowRun, error) {
			return nil, nil
		}),
		WithNewInput(func() any { return nil }),
	)
	assert.ErrorIs(t, err, ErrInvalidDefinition)
}

func TestNew_RequiresTaskQueue(t *testing.T) {
	_, err := New("name", "",
		WithRegister(func(worker.Worker) {}),
		WithExecute(func(context.Context, client.Client, client.StartWorkflowOptions, any) (client.WorkflowRun, error) {
			return nil, nil
		}),
		WithNewInput(func() any { return nil }),
	)
	assert.ErrorIs(t, err, ErrInvalidDefinition)
}

func TestNew_RequiresAllClosures(t *testing.T) {
	_, err := New("name", "tq")
	assert.ErrorIs(t, err, ErrInvalidDefinition)

	_, err = New("name", "tq", WithRegister(func(worker.Worker) {}))
	assert.ErrorIs(t, err, ErrInvalidDefinition)

	_, err = New("name", "tq",
		WithRegister(func(worker.Worker) {}),
		WithExecute(func(context.Context, client.Client, client.StartWorkflowOptions, any) (client.WorkflowRun, error) {
			return nil, nil
		}),
	)
	assert.ErrorIs(t, err, ErrInvalidDefinition)
}

func TestNew_ValidScheduleAccepted(t *testing.T) {
	d, err := New("name", "tq",
		WithRegister(func(worker.Worker) {}),
		WithExecute(func(context.Context, client.Client, client.StartWorkflowOptions, any) (client.WorkflowRun, error) {
			return nil, nil
		}),
		WithNewInput(func() any { return nil }),
		WithSchedule(&ScheduleSpec{Interval: 1}),
		WithDescription("desc"),
		WithTags("a", "b"),
	)
	require.NoError(t, err)
	assert.Equal(t, "name", d.Name)
	assert.Equal(t, "tq", d.TaskQueue)
	assert.Equal(t, "desc", d.Description)
	assert.Equal(t, []string{"a", "b"}, d.Tags)
	require.NotNil(t, d.Schedule)
}

func TestNew_InvalidScheduleRejected(t *testing.T) {
	_, err := New("name", "tq",
		WithRegister(func(worker.Worker) {}),
		WithExecute(func(context.Context, client.Client, client.StartWorkflowOptions, any) (client.WorkflowRun, error) {
			return nil, nil
		}),
		WithNewInput(func() any { return nil }),
		WithSchedule(&ScheduleSpec{}), // nothing set
	)
	assert.ErrorIs(t, err, ErrInvalidDefinition)
}

// fakeWorker is a minimal stub implementing worker.Worker — only the two
// register methods are exercised in unit tests.
type fakeWorker struct {
	workflowRegistrations int32
	activityRegistrations int32
}

func (f *fakeWorker) RegisterWorkflow(_ any) {}
func (f *fakeWorker) RegisterWorkflowWithOptions(_ any, _ workflow.RegisterOptions) {
	atomic.AddInt32(&f.workflowRegistrations, 1)
}
func (f *fakeWorker) RegisterDynamicWorkflow(_ any, _ workflow.DynamicRegisterOptions) {}
func (f *fakeWorker) RegisterActivity(_ any)                                           {}
func (f *fakeWorker) RegisterActivityWithOptions(_ any, _ activity.RegisterOptions) {
	atomic.AddInt32(&f.activityRegistrations, 1)
}
func (f *fakeWorker) RegisterDynamicActivity(_ any, _ activity.DynamicRegisterOptions) {}
func (f *fakeWorker) RegisterNexusService(_ *nexus.Service)                            {}
func (f *fakeWorker) Start() error                                                     { return nil }
func (f *fakeWorker) Run(_ <-chan interface{}) error                                   { return nil }
func (f *fakeWorker) Stop()                                                            {}

func TestRegisterWorkflowOnce_Deduplicates(t *testing.T) {
	w := &fakeWorker{}
	RegisterWorkflowOnce(w, "myWfDedup1", func() error { return nil }, workflow.RegisterOptions{Name: "myWfDedup1"})
	RegisterWorkflowOnce(w, "myWfDedup1", func() error { return nil }, workflow.RegisterOptions{Name: "myWfDedup1"})
	RegisterWorkflowOnce(w, "myWfDedup1", func() error { return nil }, workflow.RegisterOptions{Name: "myWfDedup1"})
	assert.Equal(t, int32(1), atomic.LoadInt32(&w.workflowRegistrations))
}

func TestRegisterWorkflowOnce_DifferentWorkers(t *testing.T) {
	w1 := &fakeWorker{}
	w2 := &fakeWorker{}
	RegisterWorkflowOnce(w1, "myWfMulti", func() error { return nil }, workflow.RegisterOptions{Name: "myWfMulti"})
	RegisterWorkflowOnce(w2, "myWfMulti", func() error { return nil }, workflow.RegisterOptions{Name: "myWfMulti"})
	assert.Equal(t, int32(1), atomic.LoadInt32(&w1.workflowRegistrations))
	assert.Equal(t, int32(1), atomic.LoadInt32(&w2.workflowRegistrations))
}

func TestGetRun(t *testing.T) {
	d, err := New("name", "tq",
		WithRegister(func(worker.Worker) {}),
		WithExecute(func(context.Context, client.Client, client.StartWorkflowOptions, any) (client.WorkflowRun, error) {
			return nil, nil
		}),
		WithNewInput(func() any { return nil }),
	)
	require.NoError(t, err)
	h := d.GetRun(nil, "wf-id", "run-id")
	assert.Equal(t, "wf-id", h.WorkflowID)
	assert.Equal(t, "run-id", h.RunID)
	// raw is nil because client is nil; Get(ctx, &v) returns nil.
	assert.NoError(t, h.Get(context.Background(), nil))
}
