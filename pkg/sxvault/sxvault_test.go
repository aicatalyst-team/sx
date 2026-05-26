package sxvault

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitPutAgentWritesSXVaultFormat(t *testing.T) {
	ctx := context.Background()
	remote := filepath.Join(t.TempDir(), "vault.git")
	runGit(t, "", "init", "--bare", remote)

	client, err := OpenGit(remote, GitOptions{Actor: Actor{Name: "Test Admin", Email: "test@example.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.PutAgent(ctx, AgentSpec{
		BotName:     "reviewer",
		AssetName:   "reviewer",
		Version:     "1.0.0",
		DisplayName: "Reviewer",
		Description: "Reviews pull requests.",
		Prompt:      "You are Reviewer.",
	}); err != nil {
		t.Fatal(err)
	}

	clone := filepath.Join(t.TempDir(), "clone")
	runGit(t, "", "clone", remote, clone)
	assertFileContains(t, filepath.Join(clone, "assets", "reviewer", "1.0.0", "AGENT.md"), "You are Reviewer.")
	assertFileContains(t, filepath.Join(clone, "assets", "reviewer", "1.0.0", "metadata.toml"), `type = "agent"`)
	manifest := readFile(t, filepath.Join(clone, "sx.toml"))
	for _, want := range []string{`name = "reviewer"`, `kind = "bot"`, `bot = "reviewer"`} {
		if !strings.Contains(manifest, want) {
			t.Fatalf("sx.toml missing %q:\n%s", want, manifest)
		}
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	got := readFile(t, path)
	if !strings.Contains(got, want) {
		t.Fatalf("%s missing %q:\n%s", path, want, got)
	}
}
