package summarizer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/rubiojr/gas/internal/log"
	"github.com/tj/go-naturaldate"
	"golang.org/x/oauth2"
)

type Summarizer struct {
	client      *github.Client
	since       time.Time
	repos       []string
	searchQuery string
	ctx         context.Context
	queryExtra  string
	author      string
}

type Comment struct {
	IssueNumber int
	IssueTitle  string
	RepoName    string
	RepoOwner   string
	Body        string
	Author      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	URL         string
	IsPR        bool
	IsOpen      bool
	IssueURL    string
}

type Option func(*Summarizer)

func WithAuthor(author string) Option {
	return func(s *Summarizer) {
		s.author = author
	}
}

func WithClient(client *github.Client) Option {
	return func(s *Summarizer) {
		s.client = client
	}
}

func WithFromDate(since string) Option {
	date, err := naturaldate.Parse(since, time.Now())
	if err != nil {
		return func(s *Summarizer) {
			s.since = date
		}
	}

	return func(s *Summarizer) {
		s.since = date
	}
}

func WithRepo(repo string) Option {
	return func(s *Summarizer) {
		s.repos = append(s.repos, repo)
	}
}

func WithQueryExtra(query string) Option {
	return func(s *Summarizer) {
		s.queryExtra = " " + query
	}
}

func WithSearchQuery(query string) Option {
	return func(s *Summarizer) {
		s.searchQuery = query
	}
}

func NewSummarizer(ctx context.Context, token string, opts ...Option) (*Summarizer, error) {
	// Create OAuth2 token for authentication
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create GitHub client
	client := github.NewClient(tc)

	oneWeekAgo := time.Now().AddDate(0, 0, -7)

	s := &Summarizer{client: client, since: oneWeekAgo, ctx: ctx}
	for _, opt := range opts {
		opt(s)
	}

	if s.author == "" {
		// Get authenticated user if not set
		user, _, err := client.Users.Get(s.ctx, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get authenticated user: %v", err)
		}
		s.author = *user.Login
	}

	return s, nil
}

func (s *Summarizer) GetRecentParticipationComments() ([]Comment, error) {
	query := s.searchQuery
	if query == "" {
		// Create a search query for issues and PRs you participated in
		query = fmt.Sprintf("involves:%s updated:>=%s", s.author, s.since.Format("2006-01-02"))

		for _, repo := range s.repos {
			query += fmt.Sprintf(" repo:%s", repo)
		}
	}

	if s.queryExtra != "" {
		query += fmt.Sprintf(" %s", s.queryExtra)
	}

	log.Stderr(fmt.Sprintf("query: %s", query))

	opt := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Sort:        "updated",
		Order:       "desc",
	}

	var allComments []Comment

	// Search for issues/PRs
	for {
		issues, resp, err := s.client.Search.Issues(s.ctx, query, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to search issues: %v", err)
		}

		// Process each issue/PR
		for _, issue := range issues.Issues {
			isPR := issue.PullRequestLinks != nil

			components := strings.Split(*issue.RepositoryURL, "/")
			repoName, repoOwner := components[len(components)-1], components[len(components)-2]

			// Get issue/PR comments
			var comments []*github.IssueComment
			var commentsResp *github.Response

			commentsOpt := &github.IssueListCommentsOptions{
				ListOptions: github.ListOptions{PerPage: 100},
			}

			for {
				comments, commentsResp, err = s.client.Issues.ListComments(s.ctx, repoOwner, repoName, *issue.Number, commentsOpt)
				if err != nil {
					return nil, fmt.Errorf("failed to get comments for issue #%d in %s/%s: %v",
						*issue.Number, repoOwner, repoName, err)
				}

				// Process comments
				for _, comment := range comments {
					if *comment.User.Login != s.author {
						continue
					}

					if comment.CreatedAt.Before(s.since) {
						continue
					}

					allComments = append(allComments, Comment{
						IssueNumber: *issue.Number,
						IssueTitle:  *issue.Title,
						RepoName:    repoName,
						RepoOwner:   repoOwner,
						Body:        bodyOrEmpty(issue),
						Author:      *comment.User.Login,
						CreatedAt:   comment.CreatedAt.Time,
						UpdatedAt:   comment.UpdatedAt.Time,
						URL:         *comment.HTMLURL,
						IsPR:        isPR,
						IsOpen:      *issue.State == "open",
						IssueURL:    *issue.HTMLURL,
					})
				}

				if commentsResp.NextPage == 0 {
					break
				}
				commentsOpt.Page = commentsResp.NextPage
			}

			// If it's a PR, also get review comments
			if isPR {
				reviewCommentsOpt := &github.PullRequestListCommentsOptions{
					ListOptions: github.ListOptions{PerPage: 100},
				}

				for {
					reviewComments, reviewCommentsResp, err := s.client.PullRequests.ListComments(s.ctx, repoOwner, repoName, *issue.Number, reviewCommentsOpt)
					if err != nil {
						return nil, fmt.Errorf("failed to get review comments for PR #%d in %s/%s: %v",
							*issue.Number, repoOwner, repoName, err)
					}

					// Process review comments
					for _, comment := range reviewComments {
						if *comment.User.Login != s.author {
							continue
						}

						allComments = append(allComments, Comment{
							IssueNumber: *issue.Number,
							IssueTitle:  *issue.Title,
							RepoName:    repoName,
							RepoOwner:   repoOwner,
							Body:        bodyOrEmpty(issue),
							Author:      *comment.User.Login,
							CreatedAt:   comment.CreatedAt.Time,
							UpdatedAt:   comment.UpdatedAt.Time,
							URL:         *comment.HTMLURL,
							IsPR:        true,
							IsOpen:      *issue.State == "open",
							IssueURL:    *issue.HTMLURL,
						})
					}

					if reviewCommentsResp.NextPage == 0 {
						break
					}
					reviewCommentsOpt.Page = reviewCommentsResp.NextPage
				}
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allComments, nil
}

func bodyOrEmpty(issue *github.Issue) string {
	if issue.Body == nil {
		return ""
	}
	return *issue.Body
}

func (s *Summarizer) GetOpenedPRs() ([]Comment, error) {
	author := s.author

	query := fmt.Sprintf("author:%s type:pr created:>=%s", author, s.since.Format("2006-01-02"))

	// Add specific repos if provided
	for _, repo := range s.repos {
		query += fmt.Sprintf(" repo:%s", repo)
	}

	if s.queryExtra != "" {
		query += fmt.Sprintf(" %s", s.queryExtra)
	}

	log.Stderr(fmt.Sprintf("query: %s", query))

	opt := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Sort:        "created",
		Order:       "desc",
	}

	var openedPRs []Comment

	// Search for PRs
	for {
		issues, resp, err := s.client.Search.Issues(s.ctx, query, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to search PRs: %v", err)
		}

		// Process each PR
		for _, issue := range issues.Issues {
			// Skip if not a PR
			if issue.PullRequestLinks == nil {
				continue
			}

			components := strings.Split(*issue.RepositoryURL, "/")
			repoName, repoOwner := components[len(components)-1], components[len(components)-2]

			openedPRs = append(openedPRs, Comment{
				IssueNumber: *issue.Number,
				IssueTitle:  *issue.Title,
				RepoName:    repoName,
				RepoOwner:   repoOwner,
				Body:        bodyOrEmpty(issue),
				Author:      author,
				CreatedAt:   issue.CreatedAt.Time,
				UpdatedAt:   issue.UpdatedAt.Time,
				URL:         *issue.HTMLURL,
				IsPR:        true,
				IsOpen:      *issue.State == "open",
				IssueURL:    *issue.HTMLURL,
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return openedPRs, nil
}
