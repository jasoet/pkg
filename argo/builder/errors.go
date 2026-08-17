package builder

import "errors"

// Sentinel errors returned by the workflow builder. Callers can match them with
// errors.Is to branch on failure modes without string comparison. Build() aggregates
// all accumulated errors with errors.Join, so a returned error may match more than one
// of these (and may also wrap errors from the underlying WorkflowSource implementations).
var (
	// ErrTemplateConflict is returned when two templates share the same name but differ
	// in content. The second, differing definition is dropped (the first one wins), which
	// would otherwise silently make a step run the wrong image or command.
	ErrTemplateConflict = errors.New("argo/builder: conflicting templates with the same name")

	// ErrEntrypointNotFound is returned by BuildWithEntrypoint when the named entrypoint
	// template has not been added to the builder.
	ErrEntrypointNotFound = errors.New("argo/builder: entrypoint template not found")

	// ErrTemplateSource is returned when a WorkflowSource fails to produce its steps or
	// templates during Add, AddParallel, or AddExitHandler.
	ErrTemplateSource = errors.New("argo/builder: workflow source error")
)
