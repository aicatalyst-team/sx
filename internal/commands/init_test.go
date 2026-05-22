package commands

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// TestPromptForValidGitURL covers the init-time URL validation loop.
// The git remote call is replaced by a stub verifier so we can simulate
// different GitHub responses (success, auth required, not found) without
// running git or hitting the network. This is the test that proves
// `sx init` for github actually checks the response — if anyone disconnects
// the verifier from the prompt loop, these cases fail.
func TestPromptForValidGitURL(t *testing.T) {
	ctx := context.Background()

	t.Run("accepts URL when verifier returns nil", func(t *testing.T) {
		var seen []string
		verify := func(_ context.Context, url string) error {
			seen = append(seen, url)
			return nil
		}
		in := bufio.NewReader(strings.NewReader("git@github.com:foo/bar.git\n"))
		out := &bytes.Buffer{}

		got, err := promptForValidGitURL(ctx, "", in, out, verify)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "git@github.com:foo/bar.git" {
			t.Errorf("got URL %q, want %q", got, "git@github.com:foo/bar.git")
		}
		if len(seen) != 1 || seen[0] != "git@github.com:foo/bar.git" {
			t.Errorf("verifier calls = %v, want exactly one with the entered URL", seen)
		}
	})

	t.Run("retries with new URL when verifier fails and user says yes", func(t *testing.T) {
		var seen []string
		verify := func(_ context.Context, url string) error {
			seen = append(seen, url)
			if url == "https://github.com/foo/bad" {
				return errors.New("repository not found")
			}
			return nil
		}
		// First URL fails, user confirms retry (y), second URL succeeds.
		in := bufio.NewReader(strings.NewReader("https://github.com/foo/bad\ny\nhttps://github.com/foo/good\n"))
		out := &bytes.Buffer{}

		got, err := promptForValidGitURL(ctx, "", in, out, verify)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "https://github.com/foo/good" {
			t.Errorf("got URL %q, want the second one", got)
		}
		if len(seen) != 2 {
			t.Errorf("verifier called %d times, want 2: %v", len(seen), seen)
		}
		if !strings.Contains(out.String(), "Cannot access") {
			t.Errorf("expected user-visible error, got output:\n%s", out.String())
		}
		if !strings.Contains(out.String(), "repository not found") {
			t.Errorf("verifier error not surfaced to user:\n%s", out.String())
		}
	})

	t.Run("aborts when verifier fails and user declines retry", func(t *testing.T) {
		verify := func(_ context.Context, _ string) error {
			return errors.New("authentication required")
		}
		in := bufio.NewReader(strings.NewReader("https://github.com/private/repo\nn\n"))
		out := &bytes.Buffer{}

		_, err := promptForValidGitURL(ctx, "", in, out, verify)
		if err == nil {
			t.Fatal("expected error when user declines retry")
		}
		if !strings.Contains(err.Error(), "aborted") {
			t.Errorf("got error %q, want one mentioning abort", err)
		}
		if !strings.Contains(out.String(), "authentication required") {
			t.Errorf("auth error not surfaced to user:\n%s", out.String())
		}
	})

	t.Run("rejects empty URL", func(t *testing.T) {
		called := false
		verify := func(_ context.Context, _ string) error {
			called = true
			return nil
		}
		in := bufio.NewReader(strings.NewReader("\n"))
		out := &bytes.Buffer{}

		_, err := promptForValidGitURL(ctx, "", in, out, verify)
		if err == nil {
			t.Fatal("expected error for empty URL")
		}
		if called {
			t.Error("verifier should not be called for empty URL")
		}
	})
}
