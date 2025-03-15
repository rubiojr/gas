package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rubiojr/gas/summarizer"
)

type MCPServer struct {
	server *server.MCPServer
	summ   *summarizer.Summarizer
}

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

func grabToken() string {
	token := grabTokenFromFile()
	if token == "" {
		token = grabTokenFromCLI()
	}
	return token
}

func New() (*MCPServer, error) {
	s := server.NewMCPServer(
		"GitHub Activity Summarizer",
		"1.0.0",
		server.WithPromptCapabilities(true),
	)

	token := grabTokenFromCLI()
	if token == "" {
		return nil, fmt.Errorf("Make sure you have the GitHub CLI configured")
	}

	opts := []summarizer.Option{}
	repos := os.Getenv("GITHUB_GAS_REPOSITORIES")
	if repos != "" {
		repos := strings.Split(repos, ",")
		for _, repo := range repos {
			opts = append(opts, summarizer.WithRepo(repo))
		}
	}

	queryExtra := os.Getenv("GITHUB_GAS_QUERY_EXTRA")
	if queryExtra != "" {
		opts = append(opts, summarizer.WithQueryExtra(queryExtra))
	}

	fromDate := os.Getenv("GITHUB_GAS_FROM_DATE")
	if fromDate != "" {
		opts = append(opts, summarizer.WithFromDate(fromDate))
	}

	summ, err := summarizer.NewSummarizer(
		context.Background(),
		token,
		opts...,
	)
	if err != nil {
		return nil, err
	}

	mcps := &MCPServer{server: s, summ: summ}
	prompt := mcp.NewPrompt(
		"gas",
		mcp.WithPromptDescription("Retrieves GitHub activity"),
	)
	s.AddPrompt(prompt, mcps.summarizeHandler)

	return mcps, nil
}

func (s *MCPServer) ServeStdio() error {
	return server.ServeStdio(s.server)
}

func (s *MCPServer) summarizeHandler(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	comments, err := s.summ.GetRecentParticipationComments()
	if err != nil {
		return nil, err
	}

	if len(comments) == 0 {
		return &mcp.GetPromptResult{
			Description: "No GitHub activity found",
			Messages: []mcp.PromptMessage{
				{
					Role: mcp.RoleUser,
					Content: mcp.TextContent{
						Type: "text",
						Text: "No GitHub activity found",
					},
				},
			},
		}, nil
	}

	buf := &bytes.Buffer{}
	for _, comment := range comments {
		buf.WriteString("*********************************\n")
		buf.WriteString(fmt.Sprintf("Issue URL: %s\n", comment.IssueURL))
		buf.WriteString(fmt.Sprintf("Is Pull request?: %t\n", comment.IsPR))
		buf.WriteString(fmt.Sprintf("Issue Author: %s\n", comment.Author))
		buf.WriteString(fmt.Sprintf("Comment date: %s\n", comment.CreatedAt))
		buf.WriteString(fmt.Sprintf("Comment in %s/%s #%d (%s) by %s:\n%s\n\n",
			comment.RepoOwner, comment.RepoName, comment.IssueNumber,
			comment.IssueTitle, comment.Author, comment.Body))
	}

	return &mcp.GetPromptResult{
		Description: "GitHub activity from last week",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: buf.String(),
				},
			},
		},
	}, nil
}
