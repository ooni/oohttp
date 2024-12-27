// Package testenv emulates some testenv properties to reduce
// the amount of deleted line with respect to upstream.
package testenv

import (
	"context"
	"os/exec"
	"testing"
)

// MustHaveExec always skips the current test.
func MustHaveExec(t testing.TB) {
	t.Skip("testenv.MustHaveExec is not enabled in this fork")
}

// SkipFlay skips a flaky test.
func SkipFlaky(t testing.TB, issue int) {
	t.Skip("testenv.SkipFlaky: skipping flaky test", issue)
}

// HasSrc always returns false.
func HasSrc() bool {
	return false
}

// GoToolPath returns the Go tool path.
func GoToolPath(t testing.TB) string {
	return "go"
}

// Builder always returns the empty string.
func Builder() string {
	return ""
}

func CommandContext(t testing.TB, ctx context.Context, name string, args ...string) *exec.Cmd {
	t.Skip("testenv.CommandContext is not enabled in this fork")
	return &exec.Cmd{}
}

func Command(t testing.TB, name string, args ...string) *exec.Cmd {
	return CommandContext(t, context.Background(), name, args...)
}
