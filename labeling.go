package main

import (
	"strings"
)

type labelRule struct {
	label    string
	keywords []string
}

// Rules are evaluated in order; the first keyword hit wins per rule so an
// issue ends up with at most one label per rule.
var labelRules = []labelRule{
	{"bug", []string{"bug", "crash", "panic", "broken", "fails", "exception", "regression", "segfault"}},
	{"security", []string{"security", "xss", "csrf", "injection", "privilege escalation", "auth bypass", "cve-"}},
	{"enhancement", []string{"feature", "enhance", "improve", "add support", "suggest", "would be nice", "request"}},
	{"documentation", []string{"documentation", "readme", "typo", "example", "instruction", "doc"}},
	{"question", []string{"how do", "how can", "why does", "is there a way", "usage", "?"}},
}

// categorize inspects an issue's title and body and returns the labels it
// should carry, deduplicated and in rule order.
func categorize(title, body string) []string {
	haystack := strings.ToLower(title + "\n" + body)
	var matches []string
	seen := make(map[string]bool)
	for _, rule := range labelRules {
		if seen[rule.label] {
			continue
		}
		for _, kw := range rule.keywords {
			if strings.Contains(haystack, kw) {
				matches = append(matches, rule.label)
				seen[rule.label] = true
				break
			}
		}
	}
	return matches
}

// AutoLabel tags every open issue whose text matches a rule. Returns how many
// issues were labelled (or would be, in dry-run mode).
func AutoLabel(c *Client, owner, repo string, issues []Issue, dryRun bool) (int, error) {
	added := 0
	for _, iss := range issues {
		if iss.IsPullRequest() || iss.Locked {
			continue
		}
		labels := categorize(iss.Title, iss.Body)
		if len(labels) == 0 {
			continue
		}
		var missing []string
		for _, l := range labels {
			if !iss.HasLabel(l) {
				missing = append(missing, l)
			}
		}
		if len(missing) == 0 {
			continue
		}
		added++
		if dryRun {
			continue
		}
		if err := c.AddLabels(owner, repo, iss.Number, missing); err != nil {
			return added, err
		}
	}
	return added, nil
}
