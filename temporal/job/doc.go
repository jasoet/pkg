// Package job provides a type-focused abstraction for one registered Temporal
// workflow. A Definition holds the metadata, lifecycle hooks, and per-job
// operations for a single workflow: register on a worker, execute by name,
// attach a schedule, describe runs, control lifecycle.
//
// The package coexists with pkg/temporal's existing managers (WorkflowManager,
// ScheduleManager). Use a Definition when you have a typed handle to "your"
// workflow; use the namespace-wide managers when you need to inspect
// workflows you didn't register yourself.
//
// Typical use:
//
//	def, err := job.New("orders-sync", "sync-orders",
//	    job.WithRegister(func(w worker.Worker) { /* ... */ }),
//	    job.WithExecute(func(ctx, c, opts, in) (client.WorkflowRun, error) { /* ... */ }),
//	    job.WithNewInput(func() any { return &OrdersInput{} }),
//	    job.WithSchedule(&job.ScheduleSpec{Interval: time.Hour}),
//	)
//	def.Register(worker)
//	run, _ := def.Execute(ctx, c, &OrdersInput{...})
//	detail, _ := def.Describe(ctx, c, run.WorkflowID, run.RunID)
package job
