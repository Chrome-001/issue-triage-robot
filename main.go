package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	mode := flag.String("mode", "all", "all|label|stale|report")
	repos := flag.String("repos", "", "owner/repo,owner/repo2 (or $GITHUB_REPOS)")
	days := flag.Int("days", 30, "mark stale after this many days of inactivity")
	closeDays := flag.Int("close-days", 60, "auto-close after this many days stale")
	dryRun := flag.Bool("dry-run", false, "compute only, change nothing on GitHub")
	flag.Parse()

	token := os.Getenv("GH_TOKEN")
	if token == "" {
		log.Fatal("set GH_TOKEN (repo scope) before running")
	}
	list := strings.TrimSpace(*repos)
	if list == "" {
		list = os.Getenv("GITHUB_REPOS")
	}
	if list == "" {
		log.Fatal("pass -repos owner/repo,owner/repo2 or set GITHUB_REPOS")
	}

	cfg := DefaultStaleConfig()
	cfg.StaleAfter = time.Duration(*days) * 24 * time.Hour
	cfg.CloseAfter = time.Duration(*closeDays) * 24 * time.Hour
	now := time.Now().UTC()
	client := NewClient(token)

	failed := false
	for _, pair := range strings.Split(list, ",") {
		pair = strings.TrimSpace(pair)
		parts := strings.SplitN(pair, "/", 2)
		if len(parts) != 2 {
			log.Printf("skip %q: expected owner/repo", pair)
			continue
		}
		owner, repo := parts[0], parts[1]

		issues, err := client.ListOpenIssues(owner, repo)
		if err != nil {
			log.Printf("%s: %v", pair, err)
			failed = true
			continue
		}
		log.Printf("%s: %d open issues", pair, len(issues))
		if len(issues) == 0 {
			continue
		}

		if *mode == "all" || *mode == "label" {
			n, err := AutoLabel(client, owner, repo, issues, *dryRun)
			if err != nil {
				log.Printf("label %s: %v", pair, err)
				failed = true
			} else {
				log.Printf("label %s: %d to add", pair, n)
			}
		}

		if *mode == "all" || *mode == "stale" {
			res, err := ProcessStale(client, owner, repo, issues, cfg, *dryRun, now)
			if err != nil {
				log.Printf("stale %s: %v", pair, err)
				failed = true
			} else {
				log.Printf("stale %s: %d marked, %d closed", pair, res.Marked, res.Closed)
			}
		}

		if *mode == "all" || *mode == "report" {
			report := BuildReport(owner, repo, issues, now)
			name := fmt.Sprintf("triage-%s-%s-%s.md", owner, repo, now.Format("20060102"))
			if err := os.WriteFile(name, []byte(report), 0o644); err != nil {
				log.Printf("report %s: %v", pair, err)
				failed = true
				continue
			}
			log.Printf("report %s: wrote %s", pair, name)
		}
	}

	if failed {
		os.Exit(1)
	}
}
