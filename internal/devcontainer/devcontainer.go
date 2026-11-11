// Package devcontainer edits the features and env in an o3s home's dev container config and
// keeps every comment and the layout the user gave it.
package devcontainer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"

	"github.com/hansehart/o3s/internal/atomicfile"
)

// File is where an o3s home keeps its dev container config.
const File = ".devcontainer/devcontainer.json"

// Config is a dev container config as its file holds it.
type Config struct {
	v hujson.Value
}

// Load reads the dev container config of an o3s home.
func Load(home string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(home, File))
	// Hand back a config that fails to read
	if err != nil {
		return nil, err
	}
	v, err := hujson.Parse(data)
	// Hand back a config that holds no JSON with comments
	if err != nil {
		return nil, fmt.Errorf("%s: %w", File, err)
	}
	// Turn away a config that holds no object
	if _, ok := v.Value.(*hujson.Object); !ok {
		return nil, fmt.Errorf("%s holds no object", File)
	}
	return &Config{v: v}, nil
}

// Save writes the config back into the o3s home in a single step.
func (c *Config) Save(home string) error {
	return atomicfile.Write(filepath.Join(home, File), c.v.Pack(), 0o644)
}

// Features lists the features the config names, in its order.
func (c *Config) Features() []string {
	o := c.object("features")
	var keys []string
	// Take every feature's name
	for _, m := range o.Members {
		keys = append(keys, name(m))
	}
	return keys
}

// SetFeature names a feature with its options, in the place of any tag of the same package.
func (c *Config) SetFeature(pkg, key string, options map[string]string) {
	// Write a feature without options as an empty object
	if options == nil {
		options = map[string]string{}
	}
	c.set("features", func(m hujson.ObjectMember) bool { return samePackage(name(m), pkg) }, key, options)
}

// RemoveFeature drops a feature whatever its tag and reports whether the config named it.
func (c *Config) RemoveFeature(pkg string) bool {
	return c.remove("features", func(m hujson.ObjectMember) bool { return samePackage(name(m), pkg) })
}

// SetEnv sets a variable every process in the cage runs with.
func (c *Config) SetEnv(key, value string) {
	c.set("containerEnv", func(m hujson.ObjectMember) bool { return name(m) == key }, key, value)
}

// RemoveEnv drops a variable and reports whether the config set it.
func (c *Config) RemoveEnv(key string) bool {
	return c.remove("containerEnv", func(m hujson.ObjectMember) bool { return name(m) == key })
}

// set gives the first member that matches a new name and value, or appends one in the object's layout.
func (c *Config) set(object string, match func(hujson.ObjectMember) bool, key string, value any) {
	o := c.object(object)
	indent := c.unit() + c.unit()
	v := literal(value, indent, c.unit())
	// Take the place of the member that matches
	for i, m := range o.Members {
		// Keep the comments around the member it replaces
		if match(m) {
			o.Members[i].Name.Value = hujson.String(key)
			v.BeforeExtra, v.AfterExtra = m.Value.BeforeExtra, m.Value.AfterExtra
			o.Members[i].Value = v
			return
		}
	}
	appendMember(o, key, v, indent, c.unit())
}

// remove drops the first member that matches and reports whether one did.
func (c *Config) remove(object string, match func(hujson.ObjectMember) bool) bool {
	o := c.object(object)
	// Find the member that matches
	for i, m := range o.Members {
		// Drop it, and close the object where it held nothing else
		if match(m) {
			o.Members = append(o.Members[:i], o.Members[i+1:]...)
			// Close an emptied object on the line it opens
			if len(o.Members) == 0 {
				o.AfterExtra = nil
			}
			return true
		}
	}
	return false
}

// object finds a top-level object of the config and adds it where the config lacks it.
func (c *Config) object(key string) *hujson.Object {
	root := c.v.Value.(*hujson.Object)
	// Find the object the config already holds
	for _, m := range root.Members {
		// Take the object of that name
		if o, ok := m.Value.Value.(*hujson.Object); ok && name(m) == key {
			return o
		}
	}
	o := &hujson.Object{}
	appendMember(root, key, hujson.Value{Value: o}, c.unit(), "")
	return o
}

// unit is the indent the config's top-level members carry, four spaces where it shows none.
func (c *Config) unit() string {
	root := c.v.Value.(*hujson.Object)
	// Take the indent of the first top-level member
	if len(root.Members) > 0 {
		before := string(root.Members[0].Name.BeforeExtra)
		// Measure the indent on the member's own line
		if _, indent, ok := strings.CutLast(before, "\n"); ok {
			return indent
		}
	}
	return "    "
}

// appendMember adds a member at the end of an object, on its own line at the given indent.
func appendMember(o *hujson.Object, key string, v hujson.Value, indent, unit string) {
	v.BeforeExtra = hujson.Extra(" ")
	m := hujson.ObjectMember{Name: hujson.Value{BeforeExtra: hujson.Extra("\n" + indent), Value: hujson.String(key)}, Value: v}
	// Close the object on its own line where it held nothing
	if len(o.Members) == 0 && len(o.AfterExtra) == 0 {
		o.AfterExtra = hujson.Extra("\n" + strings.TrimSuffix(indent, unit))
	}
	// Keep a trailing comma where the object ends in one
	if n := len(o.Members); n > 0 && o.Members[n-1].Value.AfterExtra != nil {
		v.AfterExtra = hujson.Extra{}
		m.Value = v
	}
	o.Members = append(o.Members, m)
}

// literal writes a value as JSON laid out at the given indent.
func literal(value any, indent, unit string) hujson.Value {
	data, err := json.MarshalIndent(value, indent, unit)
	// Fall back to an empty string for a value JSON cannot hold, which strings and string maps never are
	if err != nil {
		data = []byte(`""`)
	}
	v, err := hujson.Parse(data)
	// Fall back to an empty string for JSON that fails to parse back, which JSON from the encoder never is
	if err != nil {
		return hujson.Value{Value: hujson.String("")}
	}
	return v
}

// name reads a member's name.
func name(m hujson.ObjectMember) string {
	l, ok := m.Name.Value.(hujson.Literal)
	// Take a name that holds no string as empty
	if !ok {
		return ""
	}
	return l.String()
}

// samePackage reports whether a feature key names a package, under any tag or digest.
func samePackage(key, pkg string) bool {
	rest, ok := strings.CutPrefix(key, pkg)
	return ok && (rest == "" || rest[0] == ':' || rest[0] == '@')
}
