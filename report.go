package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// BuildReport renders a markdown triage report for one repository: label
// distribution plus the oldest untouched issues (the ones needing eyes).
func BuildReport(owner, repo string, issues []Issue, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Triage report — %s/%s\n\n", owner, repo)
	fmt.Fprintf(&b, "Generated **%s** · %d open issues (PRs excluded)\n\n",
		now.Format("2006-01-02 15:04 UTC"), len(issues))

	counts := map[string]int{}
	for _, iss := range issues {
		if len(iss.Labels) == 0 {
			counts["(unlabelled)"]++
		}
		for _, l := range iss.Labels {
			counts[l.Name]++
		}
	}

	b.WriteString("## By label\n\n")
	b.WriteString("| Label | Count |\n|---|---|\n")
	labels := make([]string, 0, len(counts))
	for l := range counts {
		labels = append(labels, l)
	}
	sort.Strings(labels)
	for _, l := range labels {
		fmt.Fprintf(&b, "| %s | %d |\n", l, counts[l])
	}

	oldest := append([]Issue(nil), issues...)
	sort.Slice(oldest, func(i, j int) bool {
		return oldest[i].UpdatedAt.Before(oldest[j].UpdatedAt)
	})
	const limit = 10
	if len(oldest) > limit {
		oldest = oldest[:limit]
	}

	b.WriteString("\n## Need attention (oldest untouched)\n\n")
	for _, iss := range oldest {
		days := int(now.Sub(iss.UpdatedAt).Hours() / 24)
		fmt.Fprintf(&b, "- **[#%d](%s)** %s — last activity %d days ago\n",
			iss.Number, iss.HTMLURL, iss.Title, days)
	}
	return b.String()
}
