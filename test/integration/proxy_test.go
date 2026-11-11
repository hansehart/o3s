//go:build integration

package integration_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hansehart/o3s/test/stacktest"
)

// The proxy reads the home's policy: every host it lets through, and the secret it injects where.
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
