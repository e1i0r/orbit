package flow

import (
	"encoding/json"
	"testing"
)

func TestBuiltinFlowsValidityProperty(t *testing.T) {
	for _, name := range []string{"task", "quick", "careful"} {
		f, err := Resolve(nil, name)
		if err != nil {
			t.Errorf("builtin flow %q failed to resolve: %v", name, err)
		}

		if len(f.Phases) == 0 {
			t.Errorf("builtin flow %q has 0 phases", name)
		}

		if f.Name != name {
			t.Errorf("builtin flow Name = %q, want %q", f.Name, name)
		}
	}
}

func FuzzFlowValidation(f *testing.F) {
	f.Add([]byte(`{"name":"test","phases":[{"name":"p1","engine":"claude","model":"sonnet"}]}`))
	f.Add([]byte(`{"name":"invalid","phases":[]}`))
	f.Add([]byte(`broken flow format`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var fl Flow
		if err := json.Unmarshal(data, &fl); err == nil {
			_ = fl.Validate() //nolint:errcheck // fuzz testing against arbitrary flow specs
		}
	})
}
