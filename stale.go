package main

import (
	"fmt"
	"time"
)

const (
	staleLabel   = "stale"
	staleComment = "👋 This issue has been inactive for a while. If it's still\n" +
		"relevant, drop a comment to keep it open — otherwise the bot will\n" +
		"close it automatically soon."
	closeComment = "✅ Closing as stale — no activity for a long time. Please reopen\n" +
		"if this is still an issue."
)

// StaleConfig controls the stale/close thresholds and wording.
type StaleConfig struct {
	StaleAfter time.Duration
	CloseAfter time.Duration
}

func DefaultStaleConfig() StaleConfig {
	return StaleConfig{
		StaleAfter: 30 * 24 * time.Hour,
		CloseAfter: 60 * 24 * time.Hour,
	}
}

// StaleResult reports what a run changed, so callers can log a summary line.
type StaleResult struct {
	Marked int
	Closed int
}

// ProcessStale marks issues with no activity for StaleAfter as stale, then
// closes the ones that passed CloseAfter. Locked, pull-request and already
// stale issues are skipped.
func ProcessStale(c *Client, owner, repo string, issues []Issue,
	cfg StaleConfig, dryRun bool, now time.Time) (StaleResult, error) {

	var res StaleResult
	for _, iss := range issues {
		if iss.IsPullRequest() || iss.Locked || iss.HasLabel(staleLabel) {
			continue
		}
		age := now.Sub(iss.UpdatedAt)
		if age >= cfg.CloseAfter {
			res.Closed++
			if dryRun {
				continue
			}
			if err := c.Comment(owner, repo, iss.Number, closeComment); err != nil {
				return res, fmt.Errorf("comment #%d: %w", iss.Number, err)
			}
			if err := c.Close(owner, repo, iss.Number); err != nil {
				return res, fmt.Errorf("close #%d: %w", iss.Number, err)
			}
			continue
		}
		if age >= cfg.StaleAfter {
			res.Marked++
			if dryRun {
				continue
			}
			if err := c.AddLabels(owner, repo, iss.Number, []string{staleLabel}); err != nil {
				return res, fmt.Errorf("stale #%d: %w", iss.Number, err)
			}
			if err := c.Comment(owner, repo, iss.Number, staleComment); err != nil {
				return res, fmt.Errorf("comment #%d: %w", iss.Number, err)
			}
		}
	}
	return res, nil
}
