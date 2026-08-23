package flow

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
)

//go:embed flows/*.json
var builtins embed.FS

// Load reads a flow from a file and checks it.
func Load(filePath string) (Flow, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return Flow{}, fmt.Errorf("read %q: %w", filePath, err)
	}
	return decode(raw, filePath)
}

// Builtin reads one of the flows shipped inside the binary.
func Builtin(name string) (Flow, error) {
	raw, err := builtins.ReadFile(path.Join("flows", name+".json"))
	if err != nil {
		return Flow{}, fmt.Errorf("no built-in flow called %q (have: %s): %w", name, strings.Join(BuiltinNames(), ", "), err)
	}
	return decode(raw, name)
}

// BuiltinNames lists the flows shipped inside the binary, in order.
func BuiltinNames() []string {
	entries, err := builtins.ReadDir("flows")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, strings.TrimSuffix(e.Name(), ".json"))
	}
	sort.Strings(names)
	return names
}

func decode(raw []byte, source string) (Flow, error) {
	var f Flow
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return Flow{}, fmt.Errorf("parse flow %q: %w", source, err)
	}
	if err := f.Validate(); err != nil {
		return Flow{}, err
	}
	return f, nil
}
