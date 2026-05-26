package vault

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/sleuth-io/sx/internal/git"
)

// Config represents the minimal configuration needed to create a vault
// This avoids circular dependency with the config package
type Config interface {
	GetType() string
	GetServerURL() string
	GetAuthToken() string
	GetRepositoryURL() string
}

// NewFromConfig creates a vault instance from configuration
// This factory function eliminates repetitive switch statements across commands
func NewFromConfig(cfg Config) (Vault, error) {
	switch cfg.GetType() {
	case "sleuth":
		return NewSleuthVault(cfg.GetServerURL(), cfg.GetAuthToken()), nil
	case "git":
		opts := []GitVaultOption(nil)
		if tok := strings.TrimSpace(cfg.GetAuthToken()); tok != "" {
			if host := gitAuthHost(cfg.GetRepositoryURL()); host != "" {
				opts = append(opts, WithGitClient(git.NewClientWithOptions(git.WithHTTPSBasicAuth(host, "x-access-token", tok))))
			}
		}
		return NewGitVaultWithOptions(cfg.GetRepositoryURL(), opts...)
	case "path":
		return NewPathVault(cfg.GetRepositoryURL())
	default:
		return nil, fmt.Errorf("unsupported vault type: %s", cfg.GetType())
	}
}

func gitAuthHost(repoURL string) string {
	u, err := url.Parse(repoURL)
	if err == nil && u.Host != "" && u.Scheme == "https" {
		return u.Host
	}
	if strings.HasPrefix(repoURL, "git@github.com:") {
		return "github.com"
	}
	return ""
}
