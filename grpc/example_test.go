package grpc_test

import (
	"fmt"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	grpcserver "github.com/jasoet/pkg/v3/grpc"
)

// ExampleNew demonstrates creating a server with the options API. It is
// compile-only: Start blocks until shutdown, so it is not called here.
func ExampleNew() {
	server, err := grpcserver.New(
		grpcserver.WithH2CMode(),
		grpcserver.WithGRPCPort("8080"),
		grpcserver.WithServiceRegistrar(func(s *grpc.Server) {
			// Register your gRPC services, e.g.:
			// pb.RegisterYourServiceServer(s, yourService)
		}),
	)
	if err != nil {
		panic(err)
	}

	// server.Start() blocks serving until Stop is called; omitted here.
	_ = server
}

// ExampleWithGatewayRegistrar demonstrates registering handlers on the gRPC
// gateway mux. It is compile-only: Start blocks until shutdown, so it is not
// called here.
func ExampleWithGatewayRegistrar() {
	server, err := grpcserver.New(
		grpcserver.WithH2CMode(),
		grpcserver.WithGRPCPort("8080"),
		grpcserver.WithServiceRegistrar(func(s *grpc.Server) {
			// pb.RegisterYourServiceServer(s, yourService)
		}),
		// The gateway mounted under the base path (default /api/v1) only
		// serves what is registered here — typically generated code:
		//   pb.RegisterYourServiceHandlerServer(ctx, mux, conn)
		grpcserver.WithGatewayRegistrar(func(mux *runtime.ServeMux) {
			_ = mux.HandlePath(http.MethodGet, "/api/v1/ping",
				func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
					_, _ = w.Write([]byte("pong"))
				})
		}),
	)
	if err != nil {
		panic(err)
	}

	// server.Start() blocks serving until Stop is called; omitted here.
	_ = server
}

func ExampleHealthManager_RegisterCheck() {
	hm := grpcserver.NewHealthManager()
	hm.RegisterCheck("database", func() grpcserver.HealthCheckResult {
		return grpcserver.HealthCheckResult{Status: grpcserver.HealthStatusUp}
	})

	fmt.Println(hm.GetOverallStatus())
	// Output: UP
}
