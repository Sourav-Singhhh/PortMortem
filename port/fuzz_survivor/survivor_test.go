package main

import (
	"flag"
	"os"
	"testing"
	"time"

	picomatch "github.com/Sourav-Singhhh/PortMortem/port"
)

var (
	reproPattern = flag.String("pattern", "", "Specific pattern to reproduce")
	reproInput   = flag.String("input", "", "Specific input to reproduce")
)

// TestRunDifferentialSurvivor executes the continuous 60s+ differential fuzz survivor test.
func TestRunDifferentialSurvivor(t *testing.T) {
	if testing.Short() || os.Getenv("RUN_SURVIVOR") == "" {
		t.Skip("Skipping continuous 60s differential fuzz survivor test (set RUN_SURVIVOR=1 to execute)")
	}

	duration := 60 * time.Second
	seed := int64(1337)

	if err := RunSurvivor(duration, seed); err != nil {
		t.Fatalf("Differential Fuzz Survivor failed: %v", err)
	}
}

// TestReproduceSurvivor allows pinpointing and debugging specific pattern/input candidates.
func TestReproduceSurvivor(t *testing.T) {
	if *reproPattern == "" || *reproInput == "" {
		t.Skip("Skipping reproduction test: -pattern and -input flags required")
	}

	m, err := picomatch.Compile(*reproPattern, nil)
	if err != nil {
		t.Fatalf("Failed to compile pattern %q: %v", *reproPattern, err)
	}

	res := m.Match(*reproInput)
	t.Logf("Reproduction Result: pattern=%q input=%q match=%v", *reproPattern, *reproInput, res)
}
