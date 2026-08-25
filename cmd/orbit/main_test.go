package main

import (
	"bytes"
	"testing"

	"github.com/e1i0r/orbit/internal/cli"
)

func TestMainRunInvocation(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Run([]string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Errorf("expected exit code 0 on --help, got %d", code)
	}
}
