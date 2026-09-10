# issue-triage-robot

A small, dependency-free (stdlib-only) GitHub bot that tidies issues for you:
**auto-labels** them by keyword, **marks stale** ones and **auto-closes** them,
and drops a weekly **markdown triage report** into the repo.

Runs `label → stale → report` in one pass, on every repo you point it at, via
cron, systemd or a GitHub Actions schedule. `--dry-run` shows you everything it
*would* do.

```
issue-triage-robot/
├── github.go        # minimal GitHub REST client (pagination, no SDK required)
├── labeling.go      # keyword -> label rules + AutoLabel
├── stale.go         # stale detect / comment / auto-close
├── report.go        # markdown triage report
├── main.go          # CLI entrypoint (flags + env)
├── main_test.go     # httptest-backed tests of the full flow
├── Makefile
└── .github/workflows/triage.yml
```

## Status

See **`triage-<owner>-<repo>-<date>.md`** in the repo root — regenerated every run.

## Usage

```bash
make build

export GH_TOKEN=ghp_xxx          # needs issues: read + write

# preview, no changes
./bin/triage -mode all -repos Chrome-001/foo,Chrome-001/bar -dry-run

# make it real
./bin/triage -mode all -repos Chrome-001/foo,Chrome-001/bar
```

Modes: `all` (default), `label`, `stale`, `report`. Thresholds:

| flag | default | meaning |
|---|---|---|
| `-days` | 30 | mark `stale` after N days of no activity |
| `-close-days` | 60 | auto-close after being stale that long |
| `-repos` | env `GITHUB_REPOS` | comma-separated `owner/repo` list |

## What it does to an issue

- **Auto-label** — matches your issue title/body against keyword rules and adds
  `bug`, `security`, `enhancement`, `documentation` or `question`. Keyword lists
  are plain slices in `labeling.go`; extend them to taste.
- **Stale** — issues untouched for `-days` get a `stale` label and a nudge
  comment. After `-close-days` they're closed with a note. Pull requests,
  locked issues and previously-staled issues are skipped.
- **Report** — label distribution table + the 10 oldest untouched issues, which
  is where readers should go first.

## Safety features

- `--dry-run`: computes everything, mutates nothing (3 lines defended by tests).
- Cannot act on PRs or locked threads.
- Never deletes; closing is always reversible via GitHub UI.

## Schedule

**cron**:

```cron
0 4 * * 1 cd /srv/triage && GH_TOKEN=… ./bin/triage -repos Chrome-001/foo
```

**GitHub Actions**: the included `weekly-triage` workflow runs Mondays at 04:00
UTC and commits the reports. Set the `TRIAGE_REPOS` repository **variable**
(`owner/repo,owner/repo2`); the built-in `GITHUB_TOKEN` has the permission it
needs.

## Tests

```bash
make test   # or: go test ./... -v
```

The test suite spins up a fake GitHub API (`httptest`) and verifies the whole
label/stale/close graph, dry-run safety, keyword rules and report rendering.

## Design note

Zero runtime dependencies by design: `net/http` + `encoding/json`. Builds
offline, runs in any base image, and has a tiny attack surface — a deliberate
contrast to SDK-heavy bots, and easy to reason about in review.

## Support

Enjoying the triage robot? A coffee keeps the issues tidy (and the bot fed).

[![Donate via PayPal](https://img.shields.io/badge/Donate-PayPal-blue?logo=paypal)](https://paypal.me/chrome001)
[![Ko-fi](https://img.shields.io/badge/Support-Ko--fi-FF5E5B?logo=ko-fi&logoColor=white)](https://ko-fi.com/chrome001)