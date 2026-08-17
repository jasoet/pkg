//go:build integration

package db

import (
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// postgresReady is the readiness strategy every Postgres test container must
// use.
//
// Waiting only on the mapped port is not enough. The postgres image runs initdb
// against a temporary server, stops it, then starts the real one — so the
// container passes a port check while the server is on its way down, and the
// connection that follows fails with "connection reset by peer" or "unexpected
// EOF" rather than a timeout. The window is invisible on an idle machine and
// opens up under CI load, which is exactly how it surfaced: two different db
// tests failing on two consecutive runs, each ~10s in, well inside the 60s
// deadline they had.
//
// The occurrence-2 log check is what closes it — the readiness line is printed
// once by the init server and once by the real one. This mirrors
// postgres.BasicWaitStrategies(), whose own source warns these tests "will be
// flaky" on macOS without the port check kept alongside it; we keep both and
// set an explicit deadline with headroom for a loaded runner.
//
// Use this for every Postgres container in the package. MySQL and MSSQL are
// unaffected: they already wait on a real query via wait.ForSQL.
func postgresReady() testcontainers.CustomizeRequestOption {
	return testcontainers.WithWaitStrategyAndDeadline(90*time.Second,
		wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		wait.ForListeningPort("5432/tcp"),
	)
}
