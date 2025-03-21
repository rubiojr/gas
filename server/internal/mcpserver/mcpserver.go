package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rubiojr/gas/internal/keyring"
	"github.com/rubiojr/gas/summarizer"
)

const Version = "1.1.0"
const Name = "GitHub Activity Summarizer"

type MCPServer struct {
	server *server.MCPServer
	summ   *summarizer.Summarizer
}

func New() (*MCPServer, error) {
	s := server.NewMCPServer(
		Name,
		Version,
		server.WithPromptCapabilities(true),
	)

	authType := os.Getenv("GITHUB_GAS_AUTH_TYPE")
	token := keyring.GitHubToken(keyring.AuthType(authType))
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

	author := os.Getenv("GITHUB_GAS_AUTHOR")
	if author != "" {
		opts = append(opts, summarizer.WithAuthor(author))
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
	wg := sync.WaitGroup{}
	wg.Add(2)

	comments := []summarizer.Comment{}
	go func() {
		defer wg.Done()
		c, err := s.summ.GetRecentParticipationComments()
		if err != nil {
			return
		}
		comments = append(comments, c...)
	}()

	prs := []summarizer.Comment{}
	go func() {
		defer wg.Done()
		prs, _ = s.summ.GetOpenedPRs()
	}()
	wg.Wait()

	comments = append(comments, prs...)

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
		buf.WriteString(fmt.Sprintf("Comment in %s/%s #%d (%s) by %s:\nBody:\n%s\n\n",
			comment.RepoOwner, comment.RepoName, comment.IssueNumber,
			comment.IssueTitle, comment.Author, comment.Body))
	}

	return &mcp.GetPromptResult{
		Description: "GitHub Activity",
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
