package handlers

import (
	"os"
	"path/filepath"
)

// Configuration directories for OpenCode.
const (
	// GlobalConfigDir is the OpenCode global config directory, relative to home.
	GlobalConfigDir = ".config/opencode"

	// ProjectConfigDir is the OpenCode project-scoped config directory.
	ProjectConfigDir = ".opencode"

	// ConfigFile is the OpenCode config filename (lives at the root of
	// either GlobalConfigDir or the project). OpenCode also accepts a
	// JSONC variant — use ConfigFilePath to pick whichever exists on
	// disk rather than joining this constant directly.
	ConfigFile = "opencode.json"

	// ConfigFileJSONC is the JSONC variant OpenCode also accepts.
	ConfigFileJSONC = "opencode.jsonc"
)

// ConfigFilePath returns the opencode config path to read or write under
// dir. If only opencode.jsonc exists (a supported OpenCode variant), that
// path is returned so sx doesn't silently create a second opencode.json
// next to the user's existing config. Otherwise it returns the .json
// path — which is also the path used when neither file exists yet.
func ConfigFilePath(dir string) string {
	jsoncPath := filepath.Join(dir, ConfigFileJSONC)
	jsonPath := filepath.Join(dir, ConfigFile)
	if _, err := os.Stat(jsoncPath); err == nil {
		if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
			return jsoncPath
		}
	}
	return jsonPath
}

// Asset subdirectory names under an OpenCode config directory.
const (
	DirSkills     = "skills"
	DirCommands   = "commands"
	DirAgents     = "agent"
	DirRules      = "rules"
	DirMCPServers = "mcp-servers"
)

// Default prompt filenames.
const (
	DefaultSkillPromptFile   = "SKILL.md"
	DefaultCommandPromptFile = "COMMAND.md"
	DefaultAgentPromptFile   = "AGENT.md"
	DefaultRulePromptFile    = "RULE.md"
)
