package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func fakeIssue(number int, updated time.Time, labels ...string) Issue {
	var ls []Label
	for _, n := range labels {
		ls = append(ls, Label{Name: n})
	}
	return Issue{
		Number:    number,
		Title:     "Title #" + fmt.Sprint(number),
		UpdatedAt: updated,
		Labels:    ls,
		HTMLURL:   fmt.Sprintf("https://github.com/o/r/issues/%d", number),
	}
}

func testServerAndClient(t *testing.T) (*Client, *httptest.Server, *[]string) {
	t.Helper()
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case strings.HasSuffix(r.URL.Path, "/issues"):
			now := time.Now().UTC()
			payload := []Issue{
				fakeIssue(1, now.Add(-40*24*time.Hour)), // stale
				fakeIssue(2, now.Add(-90*24*time.Hour)), // close
				{Number: 3, Title: "Bug: crash on startup", UpdatedAt: now,
					PullRequest: json.RawMessage("null")}, // bug, not PR
				{Number: 4, Title: "How do I use the CLI?", UpdatedAt: now,
					PullRequest: json.RawMessage(`{"url":"x"}`)}, // PR -> skip
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(payload)
		case strings.HasSuffix(r.URL.Path, "/labels"):
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/comments"):
			w.WriteHeader(http.StatusCreated)
		default: // PATCH issues/{n}
			w.WriteHeader(http.StatusOK)
		}
	}))
	c := NewClient("t")
	c.base = srv.URL
	return c, srv, &calls
}

func TestListOpenIssuesFiltersNothingButPageSize(t *testing.T) {
	c, srv, _ := testServerAndClient(t)
	defer srv.Close()

	issues, err := c.ListOpenIssues("o", "r")
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 4 {
		t.Fatalf("want 4 issues, got %d", len(issues))
	}
	if issues[3].IsPullRequest() != true {
		t.Fatal("issue 4 should be detected as a pull request")
	}
}

func TestAutoLabel(t *testing.T) {
	c, srv, calls := testServerAndClient(t)
	defer srv.Close()

	issues, _ := c.ListOpenIssues("o", "r")
	added, err := AutoLabel(c, "o", "r", issues, false)
	if err != nil {
		t.Fatal(err)
	}
	if added != 1 {
		t.Fatalf("want 1 issue labelled, got %d", added)
	}
	joined := strings.Join(*calls, " ")
	if !strings.Contains(joined, "POST /repos/o/r/issues/3/labels") {
		t.Fatalf("expected a label POST for #3, got %s", joined)
	}
}

func TestProcessStale(t *testing.T) {
	c, srv, calls := testServerAndClient(t)
	defer srv.Close()

	issues, _ := c.ListOpenIssues("o", "r")
	cfg := DefaultStaleConfig()
	res, err := ProcessStale(c, "o", "r", issues, cfg, false, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if res.Marked != 1 || res.Closed != 1 {
		t.Fatalf("want 1 marked + 1 closed, got %+v", res)
	}
	joined := strings.Join(*calls, " ")
	if !strings.Contains(joined, "/issues/1/labels") ||
		!strings.Contains(joined, "/issues/2/comments") ||
		!strings.Contains(joined, "PATCH /repos/o/r/issues/2") {
		t.Fatalf("unexpected call graph: %s", joined)
	}
}

func TestDryRunChangesNothing(t *testing.T) {
	c, srv, calls := testServerAndClient(t)
	defer srv.Close()

	issues, _ := c.ListOpenIssues("o", "r")
	cfg := DefaultStaleConfig()
	if _, err := AutoLabel(c, "o", "r", issues, true); err != nil {
		t.Fatal(err)
	}
	if _, err := ProcessStale(c, "o", "r", issues, cfg, true, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 { // only the initial ListOpenIssues call
		t.Fatalf("dry run made %d calls, want 1", len(*calls))
	}
}

func TestCategorize(t *testing.T) {
	if got := categorize("Bug: crash on boot", ""); len(got) != 1 || got[0] != "bug" {
		t.Fatalf("bug rule failed: %v", got)
	}
	if got := categorize("Feature: add dark mode", ""); got[0] != "enhancement" {
		t.Fatalf("want enhancement first, got %v", got)
	}
	if got := categorize("plain title", "no keywords here"); len(got) != 0 {
		t.Fatalf("expected no labels, got %v", got)
	}
	multi := categorize("Bug: Security fix please", "xss in parser")
	if len(multi) != 2 || multi[0] != "bug" || multi[1] != "security" {
		t.Fatalf("multi-label ordering wrong: %v", multi)
	}
}

func TestIssueMethods(t *testing.T) {
	iss := fakeIssue(1, time.Now(), "bug", "security")
	if !iss.HasLabel("bug") || iss.HasLabel("stale") {
		t.Fatal("HasLabel misbehaved")
	}
	if iss.IsPullRequest() {
		t.Fatal("plain issue detected as PR")
	}
}

func TestBuildReport(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	issues := []Issue{
		fakeIssue(1, now.Add(-45*24*time.Hour), "bug"),
		fakeIssue(2, now.Add(-2*24*time.Hour)),
	}
	rep := BuildReport("o", "r", issues, now)
	if !strings.Contains(rep, "# Triage report — o/r") {
		t.Fatal("missing header")
	}
	if !strings.Contains(rep, "| bug | 1 |") {
		t.Fatal("missing label count")
	}
	if !strings.Contains(rep, "45 days ago") {
		t.Fatal("missing oldest-issue detail")
	}
}
