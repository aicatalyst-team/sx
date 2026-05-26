package git

import (
	"net/url"
	"strings"
)

// RemoteAuthInfo describes the host and transport parsed from a git remote URL.
type RemoteAuthInfo struct {
	Host  string
	HTTPS bool
	SSH   bool
}

// ParseRemoteAuthInfo extracts the auth-relevant host and transport from a
// git remote URL. It supports HTTPS URLs, ssh:// URLs, and scp-like SSH remotes
// such as git@github.com:owner/repo.git.
func ParseRemoteAuthInfo(repoURL string) RemoteAuthInfo {
	repoURL = strings.TrimSpace(repoURL)
	u, err := url.Parse(repoURL)
	if err == nil && u.Host != "" {
		switch strings.ToLower(u.Scheme) {
		case "https":
			return RemoteAuthInfo{Host: u.Host, HTTPS: true}
		case "ssh", "git+ssh":
			return RemoteAuthInfo{Host: u.Host, SSH: true}
		}
	}
	if host := scpLikeSSHHost(repoURL); host != "" {
		return RemoteAuthInfo{Host: host, SSH: true}
	}
	return RemoteAuthInfo{}
}

// LooksLikeHTTPSRemote reports whether repoURL uses an HTTPS scheme even when
// the URL is malformed enough that ParseRemoteAuthInfo cannot extract a host.
func LooksLikeHTTPSRemote(repoURL string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(repoURL)), "https://")
}

// DefaultHTTPSAuthUsername returns a reasonable basic-auth username for a git
// HTTPS token when the caller did not configure one explicitly.
func DefaultHTTPSAuthUsername(host, explicit string) string {
	if explicit = strings.TrimSpace(explicit); explicit != "" {
		return explicit
	}
	host = strings.ToLower(strings.TrimSpace(host))
	switch {
	case host == "gitlab.com" || strings.HasSuffix(host, ".gitlab.com"):
		return "oauth2"
	default:
		return "x-access-token"
	}
}

func scpLikeSSHHost(repoURL string) string {
	if strings.Contains(repoURL, "://") {
		return ""
	}
	at := strings.IndexByte(repoURL, '@')
	colon := strings.IndexByte(repoURL, ':')
	if at <= 0 || colon <= at+1 {
		return ""
	}
	return strings.TrimSpace(repoURL[at+1 : colon])
}
