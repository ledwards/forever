package core

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The level is read once from LEVELSIM_LEVEL while the core package initializes, before any class
// package prices a spell or picks a rank, so a test at another level has to run in its own process.
// The parent re-executes the test binary with the variable set and reads the child's verdict.
func runAtLevel(t *testing.T, level string, test string) (string, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^"+test+"$", "-test.v")
	cmd.Env = append(os.Environ(), "LEVELSIM_LEVEL="+level, "LEVELSIM_LEVEL_TEST_CHILD=1")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func inLevelChild() bool {
	return os.Getenv("LEVELSIM_LEVEL_TEST_CHILD") == "1"
}

func TestCharacterLevelDefaultsToSixty(t *testing.T) {
	if inLevelChild() {
		t.Skip("child process")
	}
	if os.Getenv("LEVELSIM_LEVEL") != "" {
		t.Skip("LEVELSIM_LEVEL is set in this environment")
	}
	if CharacterLevel != 60 {
		t.Fatalf("CharacterLevel = %d without LEVELSIM_LEVEL, want 60", CharacterLevel)
	}
}

func TestCharacterLevelFromEnvironment(t *testing.T) {
	if inLevelChild() {
		if CharacterLevel != 30 {
			t.Fatalf("CharacterLevel = %d in a process started with LEVELSIM_LEVEL=30", CharacterLevel)
		}
		return
	}
	out, err := runAtLevel(t, "30", "TestCharacterLevelFromEnvironment")
	if err != nil {
		t.Fatalf("child at level 30 failed: %v\n%s", err, out)
	}
}

// An unusable value fails the process at init, naming the variable, rather than silently running
// at 60: the worker that reads it would otherwise report numbers for a level it is not at.
func TestInvalidCharacterLevelFailsAtInit(t *testing.T) {
	if inLevelChild() {
		t.Fatal("the child must not reach a test body with an invalid level")
	}
	for _, bad := range []string{"abc", "0", "61", "-5"} {
		out, err := runAtLevel(t, bad, "TestInvalidCharacterLevelFailsAtInit")
		if err == nil {
			t.Errorf("LEVELSIM_LEVEL=%q: child exited 0, want a failure at init\n%s", bad, out)
			continue
		}
		if !strings.Contains(out, "LEVELSIM_LEVEL") {
			t.Errorf("LEVELSIM_LEVEL=%q: failure does not name the variable:\n%s", bad, out)
		}
	}
}
