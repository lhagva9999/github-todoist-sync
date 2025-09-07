package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

type Client struct {
	client *github.Client
	owner  string
	repo   string
}

type Issue struct {
	ID          int64
	Number      int
	Title       string
	Body        string
	State       string
	Labels      []string
	Assignee    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	HTMLURL     string
	IsPullReq   bool
}

func NewClient(token, owner, repo string) *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	
	return &Client{
		client: github.NewClient(tc),
		owner:  owner,
		repo:   repo,
	}
}

func (c *Client) GetIssues(ctx context.Context) ([]*Issue, error) {
	opt := &github.IssueListByRepoOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allIssues []*Issue
	
	for {
		issues, resp, err := c.client.Issues.ListByRepo(ctx, c.owner, c.repo, opt)
		if err != nil {
			return nil, fmt.Errorf("chyba při získávání issues: %v", err)
		}

		for _, issue := range issues {
			allIssues = append(allIssues, c.convertIssue(issue))
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allIssues, nil
}

func (c *Client) GetIssue(ctx context.Context, number int) (*Issue, error) {
	issue, _, err := c.client.Issues.Get(ctx, c.owner, c.repo, number)
	if err != nil {
		return nil, fmt.Errorf("chyba při získávání issue #%d: %v", number, err)
	}

	return c.convertIssue(issue), nil
}

func (c *Client) UpdateIssueState(ctx context.Context, number int, state string) error {
	issueRequest := &github.IssueRequest{
		State: &state,
	}

	_, _, err := c.client.Issues.Edit(ctx, c.owner, c.repo, number, issueRequest)
	if err != nil {
		return fmt.Errorf("chyba při aktualizaci issue #%d: %v", number, err)
	}

	return nil
}

func (c *Client) convertIssue(issue *github.Issue) *Issue {
	converted := &Issue{
		ID:        issue.GetID(),
		Number:    issue.GetNumber(),
		Title:     issue.GetTitle(),
		Body:      issue.GetBody(),
		State:     issue.GetState(),
		HTMLURL:   issue.GetHTMLURL(),
		CreatedAt: issue.GetCreatedAt().Time,
		UpdatedAt: issue.GetUpdatedAt().Time,
		IsPullReq: issue.IsPullRequest(),
	}

	if issue.Assignee != nil {
		converted.Assignee = issue.Assignee.GetLogin()
	}

	for _, label := range issue.Labels {
		converted.Labels = append(converted.Labels, label.GetName())
	}

	return converted
}
