package config_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/hansehart/o3s"
	"github.com/hansehart/o3s/internal/config"
)

// theShippedSchema compiles the schema every home receives.
func theShippedSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	data, err := o3s.Stack.ReadFile(".devcontainer/o3s.schema.json")
	// Stop when the stack ships no schema
	if err != nil {
		t.Fatalf("Setup: read the shipped schema: %v", err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	// Stop when the schema holds no JSON
	if err != nil {
		t.Fatalf("Setup: read the shipped schema: %v", err)
	}
	c := jsonschema.NewCompiler()
	// Stop when the schema stays unregistered
	if err := c.AddResource("o3s.schema.json", doc); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	schema, err := c.Compile("o3s.schema.json")
	// Stop when the schema breaks the JSON Schema rules
	if err != nil {
		t.Fatalf("Setup: compile the shipped schema: %v", err)
	}
	return schema
}

// check validates a settings file against the schema.
func check(t *testing.T, schema *jsonschema.Schema, data []byte) error {
	t.Helper()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	// Stop when the settings hold no JSON
	if err != nil {
		t.Fatalf("read settings %s: %v", data, err)
	}
	return schema.Validate(doc)
}

// Every settings file o3s writes passes the schema it ships.
func TestTheSchemaAcceptsWhatO3sWrites(t *testing.T) {
	schema := theShippedSchema(t)
	configs := map[string]config.Config{"every field": anUnorderedConfig(), "a new home": config.New()}
	// Check every config in its own home
	for name, c := range configs {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			// Stop when the settings stay unwritten
			if err := c.Save(home); err != nil {
				t.Fatalf("Setup: %v", err)
			}
			data, _ := os.ReadFile(filepath.Join(home, config.File))

			err := check(t, schema, data)

			// Report a file o3s writes yet its schema turns away
			if err != nil {
				t.Errorf("schema rejects what o3s wrote:\n%s\n%v", data, err)
			}
		})
	}
}

// The schema accepts and turns away exactly the hosts o3s does.
func TestTheSchemaAgreesWithO3s(t *testing.T) {
	schema := theShippedSchema(t)
	changes := map[string]config.Change{
		"domain":                {Name: "example.com"},
		"wildcard":              {Name: "*.example.org"},
		"address":               {Name: "192.0.2.1"},
		"range":                 {Name: "198.51.100.0/24"},
		"uppercase host":        {Name: "Example.com"},
		"wildcard in a label":   {Name: "api.*.example.com"},
		"IPv6 address":          {Name: "::1"},
		"range past 32 bits":    {Name: "192.0.2.0/33"},
		"octet past 255":        {Name: "256.0.0.1"},
		"digits as top level":   {Name: "example.123"},
		"digits leading a name": {Name: "1example.com"},
		"port zero":             {Name: "example.com", Ports: []int{0}},
		"port past the range":   {Name: "example.com", Ports: []int{65536}},
		"lowercase secret":      {Name: "api.example.com", Secret: "example_token"},
		"unknown inject site":   {Name: "api.example.com", Secret: "EXAMPLE_TOKEN", Inject: []string{"cookie"}},
		"inject with a secret":  {Name: "api.example.com", Secret: "EXAMPLE_TOKEN", Inject: []string{"header", "body"}},
		"inject alone":          {Name: "api.example.com", Inject: []string{"header"}},
	}
	// Hand every change to o3s and to the schema
	for name, ch := range changes {
		t.Run(name, func(t *testing.T) {
			var c config.Config
			_, o3sErr := c.Allow(ch)
			host := map[string]any{"name": ch.Name, "ports": []int{443}, "by": []string{"user"}}
			// Carry the ports the change names
			if len(ch.Ports) > 0 {
				host["ports"] = ch.Ports
			}
			// Carry the secret the change names
			if ch.Secret != "" {
				host["secret"] = ch.Secret
			}
			// Carry the inject sites the change names
			if len(ch.Inject) > 0 {
				host["inject"] = ch.Inject
			}
			data, _ := json.Marshal(map[string]any{"stores": []any{}, "hosts": []any{host}})

			schemaErr := check(t, schema, data)

			// Require both to agree on the change
			if (o3sErr == nil) != (schemaErr == nil) {
				t.Errorf("o3s says %v, schema says %v", o3sErr, schemaErr)
			}
		})
	}
}

// The schema turns away keys o3s never writes.
func TestTheSchemaTurnsAwayUnknownKeys(t *testing.T) {
	schema := theShippedSchema(t)
	files := map[string]string{
		"top level": `{"stores": [], "hosts": [], "extra": true}`,
		"host":      `{"stores": [], "hosts": [{"name": "example.com", "ports": [443], "by": ["user"], "extra": true}]}`,
		"store":     `{"stores": [{"id": "registry.example.com/features", "extra": true}], "hosts": []}`,
	}
	// Check every place an unknown key can hide
	for name, data := range files {
		t.Run(name, func(t *testing.T) {
			err := check(t, schema, []byte(data))

			// Require the unknown key turned away
			if err == nil {
				t.Errorf("schema accepts an unknown key at the %s", name)
			}
		})
	}
}
