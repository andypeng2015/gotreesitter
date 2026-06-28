//go:build cgo && treesitter_c_parity

package cgoharness

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

const (
	parityInDockerEnv   = "GTS_PARITY_IN_DOCKER"
	parityAllowHostEnv  = "GTS_PARITY_ALLOW_HOST"
	parityGuardrailText = "cgo parity tests are container-only; use cgo_harness/docker/run_parity_in_docker.sh or set GTS_PARITY_ALLOW_HOST=1 for a focused local debug run"
)

func TestMain(m *testing.M) {
	if !parityRunningInContainer() && !parityEnvBool(parityAllowHostEnv, false) {
		fmt.Fprintln(os.Stderr, parityGuardrailText)
		os.Exit(2)
	}
	os.Exit(m.Run())
}

func parityRunningInContainer() bool {
	if parityEnvBool(parityInDockerEnv, false) {
		return true
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}
	text := string(data)
	return strings.Contains(text, "docker") ||
		strings.Contains(text, "kubepods") ||
		strings.Contains(text, "containerd")
}

func parityEnvBool(name string, def bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}
