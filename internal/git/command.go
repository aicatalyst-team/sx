package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// execGitCommand creates a git command with optional SSH key configuration
// Returns the command ready to be executed (caller must call .Run(), .Output(), or .CombinedOutput())
func execGitCommand(ctx context.Context, sshKeyPath string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)

	// Start from the parent environment and disable interactive prompts so
	// git fails fast instead of hanging on /dev/tty when credentials are
	// missing or the repo is unreachable/private.
	env := append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
	)

	if sshKeyPath != "" {
		// Validate SSH key (log warning but continue - will fail at exec time if invalid)
		if err := ValidateSSHKey(sshKeyPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}

		// Configure SSH command
		sshCmd := buildSSHCommand(sshKeyPath)
		env = append(env, "GIT_SSH_COMMAND="+sshCmd)
	}

	cmd.Env = env
	return cmd
}

// execGitCommandWithURL prepares a git command with URL conversion if needed
// If sshKeyPath is provided and URL is HTTPS, converts URL to SSH
// Returns the command, the final URL used, and any error
func execGitCommandWithURL(ctx context.Context, sshKeyPath, url string, args ...string) (*exec.Cmd, string, error) {
	finalURL := url

	// Convert HTTPS to SSH if SSH key is provided
	if sshKeyPath != "" && IsHTTPSURL(url) {
		convertedURL, err := ConvertToSSH(url)
		if err != nil {
			return nil, "", fmt.Errorf("failed to convert URL to SSH: %w", err)
		}
		finalURL = convertedURL
	}

	// Create command with the final URL appended to args
	fullArgs := append(args, finalURL)
	cmd := execGitCommand(ctx, sshKeyPath, fullArgs...)

	return cmd, finalURL, nil
}
