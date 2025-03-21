package keyring

import (
	"os"
	"os/exec"
	"strings"

	zk "github.com/zalando/go-keyring"
)

// grab the github token from gh auth token
func grabTokenFromCLI() string {
	cmd := exec.Command("gh", "auth", "token")
	tokenBytes, err := cmd.Output()
	if err != nil {
		return ""
	}
	// Trim whitespace, especially the trailing newline
	return strings.TrimSpace(string(tokenBytes))
}

// grab the token from a file
func grabTokenFromFile() string {
	tokenFile := os.ExpandEnv("${HOME}/.config/github-gas-server/token")
	tokenBytes, err := os.ReadFile(tokenFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(tokenBytes))
}

func grabTokenFromKeyring() string {
	token, err := zk.Get("github-activity-summarizer", "github-activity-summarizer")
	if err != nil {
		return ""
	}
	return token
}

type AuthType string

const AUTH_TYPE_KEYRING AuthType = "keyring"
const AUTH_TYPE_FILE AuthType = "file"
const AUTH_TYPE_CLI AuthType = "cli"
const AUTH_TYPE_ALL AuthType = "all"

func GitHubToken(authType AuthType) string {
	var token string
	switch authType {
	case AUTH_TYPE_KEYRING:
		token = grabTokenFromKeyring()
		if token != "" {
			return token
		}
	case AUTH_TYPE_FILE:
		token = grabTokenFromFile()
		if token != "" {
			return token
		}
	case AUTH_TYPE_CLI:
		token = grabTokenFromCLI()
		if token != "" {
			return token
		}
	default:
		token = grabTokenFromKeyring()
		if token != "" {
			return token
		}
		token = grabTokenFromFile()
		if token == "" {
			token = grabTokenFromCLI()
		}
	}

	return token
}
