//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/internal/secrets"
	"github.com/hansehart/o3s/test/stacktest"
)

// The proxy reads every host of the home's policy and the secret it injects for each.
func TestTheProxyReadsTheHomesPolicy(t *testing.T) {
	policy := `{
  "$schema": "./.o3s/o3s.schema.json",
  "stores": [],
  "hosts": [
    {"name": "example.com", "ports": [22, 443], "by": ["user"]},
    {"name": "api.example.com", "ports": [443], "secret": "EXAMPLE_TOKEN", "inject": ["header", "body"], "by": ["user"]},
    {"name": "*.example.org", "ports": [443], "by": ["user"]}
  ]
}`

	got, err := stacktest.ProxyReads(t, policy, "EXAMPLE_TOKEN=s3cr3t\n")

	// Stop when the proxy fails on the policy
	if err != nil {
		t.Fatalf("ProxyReads(policy) = %v", err)
	}
	token := "s3cr3t"
	want := stacktest.Parsed{
		Keys:     []string{".example.org", "api.example.com", "example.com"},
		Allowed:  []string{".example.org", "api.example.com", "example.com"},
		Injected: map[string]*string{"api.example.com": &token},
		Sites:    map[string][]string{"api.example.com": {"body", "header"}},
	}
	// Require the hosts, the secret and its sites as the policy names them
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ProxyReads(policy) (-want +got):\n%s", diff)
	}
}

// The proxy injects the value o3s stores, in a file that keeps the empty slot the start leaves.
func TestTheProxyReadsTheSecretsO3sStores(t *testing.T) {
	path := filepath.Join(t.TempDir(), secrets.File)
	// Stop when the seeded file stays unwritten
	if err := os.WriteFile(path, []byte("# real API tokens for the egress proxy\nEXAMPLE_TOKEN=\n"), 0o600); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	store, err := secrets.Load(path)
	// Stop when the seeded file stays unread
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	// Stop when the value is turned away
	if _, err := store.Set("EXAMPLE_TOKEN", "s3cr3t"); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	policy := `{"hosts": [{"name": "api.example.com", "ports": [443], "secret": "EXAMPLE_TOKEN", "by": ["user"]}]}`

	got, err := stacktest.ProxyReads(t, policy, store.String())

	// Stop when the proxy fails on the stored secrets
	if err != nil {
		t.Fatalf("ProxyReads(policy, stored secrets) = %v", err)
	}
	token := "s3cr3t"
	// Require the stored value injected for the host that carries it
	if diff := cmp.Diff(map[string]*string{"api.example.com": &token}, got.Injected); diff != "" {
		t.Errorf("ProxyReads injected (-want +got):\n%s", diff)
	}
}
