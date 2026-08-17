package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// inspectWithNetworking builds an inspect response with the given port bindings
// and networks, mirroring what the daemon returns for a normally networked
// container.
func inspectWithNetworking(ports nat.PortMap, networks map[string]*network.EndpointSettings) container.InspectResponse {
	settings := &container.NetworkSettings{Networks: networks}
	settings.Ports = ports
	return container.InspectResponse{NetworkSettings: settings}
}

// inspectWithoutNetworking builds an inspect response with a nil
// NetworkSettings, which the daemon returns for containers created with
// --network=none and for some podman responses.
func inspectWithoutNetworking() container.InspectResponse {
	return container.InspectResponse{}
}

func TestPortBinding(t *testing.T) {
	t.Run("returns the host port for a bound container port", func(t *testing.T) {
		inspect := inspectWithNetworking(nat.PortMap{
			"80/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "32768"}},
		}, nil)

		hostPort, ok := portBinding(inspect, "80/tcp")
		assert.True(t, ok)
		assert.Equal(t, "32768", hostPort)
	})

	t.Run("reports not found for an unbound container port", func(t *testing.T) {
		inspect := inspectWithNetworking(nat.PortMap{
			"80/tcp": []nat.PortBinding{{HostPort: "32768"}},
		}, nil)

		_, ok := portBinding(inspect, "443/tcp")
		assert.False(t, ok)
	})

	t.Run("reports not found when the port has no bindings", func(t *testing.T) {
		inspect := inspectWithNetworking(nat.PortMap{"80/tcp": nil}, nil)

		_, ok := portBinding(inspect, "80/tcp")
		assert.False(t, ok)
	})

	t.Run("does not panic when the container has no networking", func(t *testing.T) {
		_, ok := portBinding(inspectWithoutNetworking(), "80/tcp")
		assert.False(t, ok)
	})
}

func TestAllPortBindings(t *testing.T) {
	t.Run("maps every bound port to its host port", func(t *testing.T) {
		inspect := inspectWithNetworking(nat.PortMap{
			"80/tcp":   []nat.PortBinding{{HostPort: "8080"}},
			"443/tcp":  []nat.PortBinding{{HostPort: "8443"}},
			"9000/tcp": nil, // exposed but unbound
		}, nil)

		assert.Equal(t, map[string]string{"80/tcp": "8080", "443/tcp": "8443"},
			allPortBindings(inspect))
	})

	t.Run("returns an empty map when the container has no networking", func(t *testing.T) {
		ports := allPortBindings(inspectWithoutNetworking())
		require.NotNil(t, ports, "callers range over the result; it must never be nil")
		assert.Empty(t, ports)
	})
}

func TestNetworkNames(t *testing.T) {
	t.Run("lists every attached network", func(t *testing.T) {
		inspect := inspectWithNetworking(nil, map[string]*network.EndpointSettings{
			"bridge":  {IPAddress: "172.17.0.2"},
			"backend": {IPAddress: "172.18.0.2"},
		})

		assert.ElementsMatch(t, []string{"bridge", "backend"}, networkNames(inspect))
	})

	t.Run("returns an empty slice when the container has no networking", func(t *testing.T) {
		names := networkNames(inspectWithoutNetworking())
		require.NotNil(t, names, "callers range over the result; it must never be nil")
		assert.Empty(t, names)
	})
}

func TestNetworkIPAddress(t *testing.T) {
	t.Run("returns the IP for the named network", func(t *testing.T) {
		inspect := inspectWithNetworking(nil, map[string]*network.EndpointSettings{
			"backend": {IPAddress: "172.18.0.2"},
		})

		ip, ok := networkIPAddress(inspect, "backend")
		assert.True(t, ok)
		assert.Equal(t, "172.18.0.2", ip)
	})

	t.Run("reports not found for an unknown network", func(t *testing.T) {
		inspect := inspectWithNetworking(nil, map[string]*network.EndpointSettings{
			"backend": {IPAddress: "172.18.0.2"},
		})

		_, ok := networkIPAddress(inspect, "frontend")
		assert.False(t, ok)
	})

	t.Run("returns the first non-empty IP when no network is named", func(t *testing.T) {
		inspect := inspectWithNetworking(nil, map[string]*network.EndpointSettings{
			"backend": {IPAddress: "172.18.0.2"},
		})

		ip, ok := networkIPAddress(inspect, "")
		assert.True(t, ok)
		assert.Equal(t, "172.18.0.2", ip)
	})

	t.Run("skips networks without an IP", func(t *testing.T) {
		inspect := inspectWithNetworking(nil, map[string]*network.EndpointSettings{
			"none": {IPAddress: ""},
		})

		_, ok := networkIPAddress(inspect, "")
		assert.False(t, ok)
	})

	t.Run("does not panic when the container has no networking", func(t *testing.T) {
		_, ok := networkIPAddress(inspectWithoutNetworking(), "bridge")
		assert.False(t, ok)

		_, ok = networkIPAddress(inspectWithoutNetworking(), "")
		assert.False(t, ok)
	})

	t.Run("nil endpoint settings are skipped rather than dereferenced", func(t *testing.T) {
		inspect := inspectWithNetworking(nil, map[string]*network.EndpointSettings{
			"bridge": nil,
		})

		_, ok := networkIPAddress(inspect, "bridge")
		assert.False(t, ok)

		_, ok = networkIPAddress(inspect, "")
		assert.False(t, ok)
	})
}
