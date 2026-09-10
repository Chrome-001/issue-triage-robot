package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const perPage = 100

// Client is a minimal GitHub REST client (stdlib only).
type Client struct {
	base   string
	token  string
	client *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		base:   "https://api.github.com",
		token:  token,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) repoURL(owner, repo, suffix string) string {
	return fmt.Sprintf("%s/repos/%s/%s/%s", c.base, owner, repo, suffix)
}

func (c *Client) newRequest(method, url string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "issue-triage-robot")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("github %s %s: %d %s",
			req.Method, req.URL.Path, resp.StatusCode, body)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Label is a single GitHub issue label.
type Label struct {
	Name string `json:"name"`
}

// Issue mirrors the fields of a GitHub issue we care about.
type Issue struct {
	Number      int             `json:"number"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	State       string          `json:"state"`
	Labels      []Label         `json:"labels"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	HTMLURL     string          `json:"html_url"`
	Locked      bool            `json:"locked"`
	PullRequest json.RawMessage `json:"pull_request,omitempty"`
}

func (i Issue) IsPullRequest() bool {
	return len(i.PullRequest) > 0 && string(i.PullRequest) != "null"
}

func (i Issue) HasLabel(name string) bool {
	for _, l := range i.Labels {
		if l.Name == name {
			return true
		}
	}
	return false
}

// ListOpenIssues returns every open issue (pull requests excluded per the
// GitHub API quirk that they appear in /issues) with full pagination.
func (c *Client) ListOpenIssues(owner, repo string) ([]Issue, error) {
	var issues []Issue
	for page := 1; ; page++ {
		url := fmt.Sprintf("%s?state=open&per_page=%d&page=%d",
			c.repoURL(owner, repo, "issues"), perPage, page)
		req, err := c.newRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		var batch []Issue
		if err := c.do(req, &batch); err != nil {
			return nil, err
		}
		issues = append(issues, batch...)
		if len(batch) < perPage {
			break
		}
	}
	return issues, nil
}

// AddLabels attaches labels to an issue, creating them if needed.
func (c *Client) AddLabels(owner, repo string, number int, labels []string) error {
	url := c.repoURL(owner, repo, fmt.Sprintf("issues/%d/labels", number))
	req, err := c.newRequest(http.MethodPost, url, map[string]any{"labels": labels})
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// Comment posts a comment on an issue.
func (c *Client) Comment(owner, repo string, number int, body string) error {
	url := c.repoURL(owner, repo, fmt.Sprintf("issues/%d/comments", number))
	req, err := c.newRequest(http.MethodPost, url, map[string]string{"body": body})
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// Close transitions an issue to the closed state.
func (c *Client) Close(owner, repo string, number int) error {
	url := c.repoURL(owner, repo, fmt.Sprintf("issues/%d", number))
	req, err := c.newRequest(http.MethodPatch, url, map[string]string{"state": "closed"})
	if err != nil {
		return err
	}
	return c.do(req, nil)
}
