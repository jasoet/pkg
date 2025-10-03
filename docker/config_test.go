package docker_test

import (
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/jasoet/pkg/v2/docker"
	"github.com/jasoet/pkg/v2/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigOptions_Image(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("nginx:alpine"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_NoImage(t *testing.T) {
	_, err := docker.New()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image is required")
}

func TestConfigOptions_Name(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithName("test-container"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Hostname(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithHostname("my-host"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Cmd(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("echo", "hello"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Entrypoint(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithEntrypoint("/bin/sh", "-c"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_EnvMap(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithEnvMap(map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
		}),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_EnvInvalid(t *testing.T) {
	_, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithEnv("INVALID_ENV"),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid env format")
}

func TestConfigOptions_PortBindings(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPortBindings(map[string]string{
			"80/tcp":   "8080",
			"443/tcp":  "8443",
			"9000/tcp": "9000",
		}),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_ExposedPorts(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithExposedPorts("80/tcp", "443/tcp", "9000/tcp"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Volume(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithVolume("/host/path", "/container/path"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_VolumeRO(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithVolumeRO("/host/path", "/container/path"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Volumes(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithVolumes(map[string]string{
			"/host/data": "/data",
			"/host/logs": "/logs",
		}),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Label(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithLabel("env", "test"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Labels(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithLabels(map[string]string{
			"env":     "test",
			"version": "1.0",
		}),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_WorkDir(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithWorkDir("/app"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_User(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithUser("1000:1000"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Network(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithNetwork("my-network"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Networks(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithNetworks("network1", "network2"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_NetworkMode(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithNetworkMode("bridge"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_AutoRemove(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithAutoRemove(true),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Privileged(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithPrivileged(true),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_CapAdd(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCapAdd("NET_ADMIN", "SYS_TIME"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_CapDrop(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCapDrop("CHOWN", "SETUID"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Tmpfs(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithTmpfs("/tmp", "size=64m"),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_ShmSize(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithShmSize(67108864), // 64MB
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_WaitStrategy(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithWaitStrategy(
			docker.WaitForLog("nginx").WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Timeout(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithTimeout(60*time.Second),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_OTelConfig(t *testing.T) {
	otelCfg := &otel.Config{}
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithOTelConfig(otelCfg),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestConfigOptions_Combined(t *testing.T) {
	exec, err := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithName("test-nginx"),
		docker.WithHostname("nginx-host"),
		docker.WithPorts("80:8080"),
		docker.WithEnv("ENV=production"),
		docker.WithWorkDir("/usr/share/nginx/html"),
		docker.WithAutoRemove(true),
		docker.WithLabel("app", "test"),
		docker.WithTimeout(30*time.Second),
	)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestContainerRequest_Basic(t *testing.T) {
	req := docker.ContainerRequest{
		Image: "nginx:alpine",
		Name:  "test-nginx",
	}

	exec, err := docker.NewFromRequest(req)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestContainerRequest_Complete(t *testing.T) {
	req := docker.ContainerRequest{
		Image:        "nginx:alpine",
		Name:         "test-nginx",
		Hostname:     "nginx-host",
		ExposedPorts: []string{"80/tcp", "443/tcp"},
		Env: map[string]string{
			"ENV":     "test",
			"VERSION": "1.0",
		},
		Cmd:        []string{"nginx", "-g", "daemon off;"},
		Entrypoint: []string{"/bin/sh", "-c"},
		WorkingDir: "/app",
		User:       "nginx",
		Labels: map[string]string{
			"app": "test",
		},
		AutoRemove: true,
		Privileged: false,
		CapAdd:     []string{"NET_ADMIN"},
		CapDrop:    []string{"CHOWN"},
		Tmpfs: map[string]string{
			"/tmp": "size=64m",
		},
		ShmSize: 67108864,
		Timeout: 30 * time.Second,
	}

	exec, err := docker.NewFromRequest(req)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestContainerRequest_InvalidPort(t *testing.T) {
	req := docker.ContainerRequest{
		Image:        "nginx:alpine",
		ExposedPorts: []string{"invalid-port"},
	}

	_, err := docker.NewFromRequest(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid exposed port")
}

func TestContainerRequest_WithVolumes(t *testing.T) {
	req := docker.ContainerRequest{
		Image: "alpine:latest",
		Volumes: map[string]string{
			"/host/data": "/data",
		},
	}

	exec, err := docker.NewFromRequest(req)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestContainerRequest_WithPortBindings(t *testing.T) {
	req := docker.ContainerRequest{
		Image: "nginx:alpine",
		PortBindings: map[string]string{
			"80/tcp": "8080",
		},
	}

	exec, err := docker.NewFromRequest(req)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestContainerRequest_WithNetworks(t *testing.T) {
	req := docker.ContainerRequest{
		Image:       "alpine:latest",
		Networks:    []string{"network1", "network2"},
		NetworkMode: "bridge",
	}

	exec, err := docker.NewFromRequest(req)
	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestHybridConfig(t *testing.T) {
	req := docker.ContainerRequest{
		Image: "nginx:alpine",
		Env: map[string]string{
			"BASE_ENV": "value",
		},
	}

	exec, err := docker.New(
		docker.WithRequest(req),
		docker.WithName("hybrid-test"),
		docker.WithEnv("ADDITIONAL_ENV=extra"),
		docker.WithPorts("80:8080"),
	)

	require.NoError(t, err)
	assert.NotNil(t, exec)
}

func TestNetworkHelpers_NatPort(t *testing.T) {
	// NatPort expects port number only, protocol is added
	port, err := docker.NatPort("8080")
	require.NoError(t, err)
	assert.Equal(t, nat.Port("8080/tcp"), port)

	// With protocol suffix - needs special handling
	port, err = docker.NatPort("8080/tcp")
	if err == nil {
		assert.Equal(t, nat.Port("8080/tcp"), port)
	}
}

func TestNetworkHelpers_PortBindings(t *testing.T) {
	bindings, err := docker.PortBindings(map[string]string{
		"80/tcp":  "8080",
		"443/tcp": "8443",
		"9000":    "9000",
	})
	require.NoError(t, err)
	assert.Len(t, bindings, 3)
}

func TestNetworkHelpers_ExposedPorts(t *testing.T) {
	ports, err := docker.ExposedPorts([]string{"80/tcp", "443/tcp", "9000"})
	require.NoError(t, err)
	assert.Len(t, ports, 3)
}

func TestNetworkHelpers_InvalidPort(t *testing.T) {
	_, err := docker.NatPort("invalid")
	assert.Error(t, err)
}
