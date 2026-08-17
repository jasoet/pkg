package docker

import (
	"testing"
	"time"
)

func TestDeriveHost(t *testing.T) {
	tests := []struct {
		name       string
		daemonHost string
		want       string
	}{
		{"empty falls back to localhost", "", "localhost"},
		{"unix socket falls back to localhost", "unix:///var/run/docker.sock", "localhost"},
		{"npipe falls back to localhost", "npipe:////./pipe/docker_engine", "localhost"},
		{"tcp remote returns host", "tcp://192.168.1.10:2375", "192.168.1.10"},
		{"tcp hostname returns host", "tcp://docker.example.com:2376", "docker.example.com"},
		{"ssh remote returns host", "ssh://user@remote-host", "remote-host"},
		{"unparseable falls back to localhost", "::::not a url", "localhost"},
		{"tcp localhost stays localhost", "tcp://localhost:2375", "localhost"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveHost(tt.daemonHost); got != tt.want {
				t.Errorf("deriveHost(%q) = %q, want %q", tt.daemonHost, got, tt.want)
			}
		})
	}
}

func TestStopTimeoutSeconds(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want int
	}{
		{"zero", 0, 0},
		{"negative", -5 * time.Second, 0},
		{"exact one second", time.Second, 1},
		{"sub-second rounds up to one", 500 * time.Millisecond, 1},
		{"tiny sub-second rounds up to one", time.Millisecond, 1},
		{"thirty seconds", 30 * time.Second, 30},
		{"1.5 seconds rounds up to two", 1500 * time.Millisecond, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stopTimeoutSeconds(tt.d); got != tt.want {
				t.Errorf("stopTimeoutSeconds(%v) = %d, want %d", tt.d, got, tt.want)
			}
		})
	}
}
