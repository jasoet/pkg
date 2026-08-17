package grpc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"

	pkgotel "github.com/jasoet/pkg/v3/otel"
)

// slowHealthServer answers Check after sleeping for delay, simulating a
// long-running unary RPC.
type slowHealthServer struct {
	grpchealth.UnimplementedHealthServer
	delay time.Duration
}

func (s *slowHealthServer) Check(context.Context, *grpchealth.HealthCheckRequest) (*grpchealth.HealthCheckResponse, error) {
	time.Sleep(s.delay)
	return &grpchealth.HealthCheckResponse{Status: grpchealth.HealthCheckResponse_SERVING}, nil
}

// blockingHealthServer signals `entered` then blocks in Check until `release`
// is closed, letting a test hold an RPC in-flight across a shutdown.
type blockingHealthServer struct {
	grpchealth.UnimplementedHealthServer
	entered chan struct{}
	release chan struct{}
}

func (s *blockingHealthServer) Check(ctx context.Context, _ *grpchealth.HealthCheckRequest) (*grpchealth.HealthCheckResponse, error) {
	select {
	case s.entered <- struct{}{}:
	default:
	}
	// Block until explicitly released, but also honor context cancellation like a
	// real handler must. GracefulStop does not cancel the RPC context, so the
	// handler stays blocked and graceful shutdown times out; the forced Stop that
	// follows cancels the context, letting the handler (and Stop) return instead
	// of deadlocking.
	select {
	case <-s.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return &grpchealth.HealthCheckResponse{Status: grpchealth.HealthCheckResponse_SERVING}, nil
}

// TestH2CLongRPCNotKilledByWriteTimeout is the end-to-end regression test for
// the critical H2C bug: in H2C mode the *http.Server's Read/WriteTimeout must
// NOT be inherited as per-stream HTTP/2 deadlines, or any RPC slower than the
// (small) write timeout is aborted with RST_STREAM. It starts a real H2C server
// with a tiny WriteTimeout, runs a real gRPC call whose handler runs much
// longer, and asserts the call still succeeds.
func TestH2CLongRPCNotKilledByWriteTimeout(t *testing.T) {
	port := freePort(t)

	server, err := New(
		WithH2CMode(),
		WithGRPCPort(port),
		WithReadTimeout(300*time.Millisecond),
		WithWriteTimeout(300*time.Millisecond),
		WithShutdownTimeout(5*time.Second),
		WithServiceRegistrar(func(s *grpc.Server) {
			grpchealth.RegisterHealthServer(s, &slowHealthServer{delay: 1200 * time.Millisecond})
		}),
	)
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- server.Start() }()
	t.Cleanup(func() {
		_ = server.Stop()
		<-startErr
	})

	waitForPort(t, port, 5*time.Second)

	conn, err := grpc.NewClient("127.0.0.1:"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := grpchealth.NewHealthClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Check(ctx, &grpchealth.HealthCheckRequest{})
	require.NoError(t, err, "a long RPC must not be killed by the H2C write timeout")
	assert.Equal(t, grpchealth.HealthCheckResponse_SERVING, resp.GetStatus())
}

// TestStopGracefulTimeoutReturnsError verifies that when graceful shutdown does
// not complete within shutdownTimeout and the server force-stops, Stop reports
// an error instead of masking the forced kill as a clean shutdown.
func TestStopGracefulTimeoutReturnsError(t *testing.T) {
	grpcPort := freePort(t)
	httpPort := freePort(t)

	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	var closeOnce sync.Once
	t.Cleanup(func() { closeOnce.Do(func() { close(release) }) })

	server, err := New(
		WithSeparateMode(grpcPort, httpPort),
		WithShutdownTimeout(300*time.Millisecond),
		WithServiceRegistrar(func(s *grpc.Server) {
			grpchealth.RegisterHealthServer(s, &blockingHealthServer{entered: entered, release: release})
		}),
	)
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- server.Start() }()
	waitForPort(t, grpcPort, 5*time.Second)

	conn, err := grpc.NewClient("127.0.0.1:"+grpcPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := grpchealth.NewHealthClient(conn)
	callDone := make(chan struct{})
	go func() {
		defer close(callDone)
		_, _ = client.Check(context.Background(), &grpchealth.HealthCheckRequest{})
	}()

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("health handler never entered")
	}

	stopErr := server.Stop()
	require.Error(t, stopErr, "Stop must report an error when graceful shutdown times out and forces stop")

	// Release the handler and let Start return.
	closeOnce.Do(func() { close(release) })
	<-callDone
	err = recvWithTimeout(t, startErr, 10*time.Second)
	assert.NoError(t, err)
}

// countingMeterProvider / countingMeter wrap real (no-op) instances and count
// how many times a Float64ObservableGauge is created, so a test can prove that
// server observable gauges are registered exactly once across restarts.
type countingMeterProvider struct {
	metric.MeterProvider
	count *int
}

func (p *countingMeterProvider) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	return &countingMeter{Meter: p.MeterProvider.Meter(name, opts...), count: p.count}
}

type countingMeter struct {
	metric.Meter
	count *int
}

func (m *countingMeter) Float64ObservableGauge(name string, opts ...metric.Float64ObservableGaugeOption) (metric.Float64ObservableGauge, error) {
	*m.count++
	return m.Meter.Float64ObservableGauge(name, opts...)
}

// TestServerMetricsRegisteredOnceAcrossRestart verifies that rebuilding the gRPC
// server (as a Start after Stop does) does not re-register the server uptime /
// start_time observable gauges, which would leave duplicate callbacks emitting
// conflicting values.
func TestServerMetricsRegisteredOnceAcrossRestart(t *testing.T) {
	count := 0
	mp := &countingMeterProvider{MeterProvider: metricnoop.NewMeterProvider(), count: &count}
	cfg := pkgotel.NewConfig("test", pkgotel.WithMeterProvider(mp))

	server, err := New(
		WithH2CMode(),
		WithGRPCPort(freePort(t)),
		WithOTelConfig(cfg),
	)
	require.NoError(t, err)

	// New() ran setupGRPCServer once: uptime + start_time => 2 registrations.
	require.Equal(t, 2, count, "expected exactly two observable gauges registered on first setup")

	// Simulate the restart rebuild path (Start after Stop nils grpcServer).
	server.grpcServer = nil
	server.setupGRPCServer()

	assert.Equal(t, 2, count, "observable gauges must not be re-registered on restart")
}
