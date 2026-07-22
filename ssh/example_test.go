package ssh_test

import (
	"fmt"

	"github.com/jasoet/pkg/v3/otel"
	"github.com/jasoet/pkg/v3/ssh"
)

// ExampleNew shows how to assemble a Config and pass functional options.
// It does not dial: New only builds the Tunnel. Secrets are injected from
// the environment because Password/PrivateKey/PrivateKeyPassphrase are
// tagged yaml:"-" and cannot come from config files.
func ExampleNew() {
	tunnel := ssh.New(ssh.Config{
		Host:       "bastion.example.com",
		Port:       22,
		User:       "deploy",
		Password:   "from-env", // e.g. os.Getenv("SSH_PASSWORD")
		RemoteHost: "postgres.internal",
		RemotePort: 5432,
		LocalPort:  15432,
	}, ssh.WithOTelConfig(otel.NewConfig("my-service")))

	// LocalAddr reports the bound address once Start has run; before that it
	// is empty.
	fmt.Println(tunnel.LocalAddr() == "")
	// Output: true
}
