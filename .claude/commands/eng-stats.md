# Engineering Team Stats

Generate PR and contribution statistics for the engineering team.

**Note**: This command is work-specific. Configuration is stored in `loyal/config/eng-stats.yaml`.

## Arguments

- `$ARGUMENTS` - Optional date range selector:
  - Time period string (default: `"1 month"`), e.g. `"2 weeks"`, `"10 days"`, `"3 months"` - interpret as "last N units from today"
  - Specific date in `YYYY-MM-DD` format, e.g. `"2025-11-17"` - interpret as start date, with the range from that date up to today
  - If `$ARGUMENTS` is missing or in an unrecognized format, default to "last 1 month from today"

## Configuration

Read configuration from `loyal/config/eng-stats.yaml`:

```yaml
org: Cellular-Longevity
repos:
  - della
  - loyal-website
  - jumbo
  - jumbo-k8s-production
exclude_authors:
  - jessgr
  - eAndrewscsg
pagination:
  page_size: 100
  max_pages: 10
```

If the config file is missing:

1. Display error: "Configuration file not found: loyal/config/eng-stats.yaml"
2. Suggest: "Copy from loyal/config/eng-stats.yaml.example or create with required fields: org, repos, exclude_authors, pagination"
3. Exit without attempting to run queries

## Instructions

1. **Load configuration** from `loyal/config/eng-stats.yaml`:
   - Parse the YAML file to get ORG, REPOS, EXCLUDE_AUTHORS, and pagination settings
   - If file doesn't exist, display error: "Configuration file not found: loyal/config/eng-stats.yaml"

2. **Calculate the date range** and `START_DATE` from `$ARGUMENTS`:
   - Use "today" as the current date in the local timezone
   - If `$ARGUMENTS` is empty, default to **1 calendar month before today** (e.g., Dec 17 - Nov 17)
   - If `$ARGUMENTS` is a relative period like `"2 weeks"` or `"3 months"`, subtract that period from today
   - If `$ARGUMENTS` is an absolute date like `"2025-11-17"`, use that directly as `START_DATE`
   - Format `START_DATE` as ISO 8601: `YYYY-MM-DD`

3. **Fetch merged PRs to main branch** for all repos in parallel:

   ```bash
   for REPO in ${REPOS[@]}; do
     gh pr list --repo {ORG}/$REPO --state merged --base main --limit 100 \
       --json author,additions,deletions,number,title \
       --search "merged:>={START_DATE}"
   done
   ```

   (Run these concurrently to minimize API wait time)

4. **Calculate per-author stats**:
   - PRs: count of merged PRs
   - LOC Changed: additions + deletions
   - Avg/PR: LOC Changed / PRs

5. **Fetch review/comment stats** using the Python pagination helper:

   Use the Python helper script to handle GraphQL pagination (avoids shell variable persistence issues):

   ```bash
   python3 ~/git/dotfiles/tools/gh-helpers/eng_stats.py --start-date {START_DATE} --config ~/git/dotfiles/loyal/config/eng-stats.yaml
   ```

   The script handles:
   - Cursor-based pagination with configurable page size and max pages
   - Early termination when encountering PRs older than START_DATE
   - Author exclusion based on configuration
   - Aggregated review comments and approvals per author

   Parse the JSON output to extract `review_stats` for each author:
   - `comments`: Total review comments
   - `approvals`: Number of APPROVED reviews

6. **Exclude authors** listed in EXCLUDE_AUTHORS from final stats:
   - Match GitHub login names case-insensitively
   - Exclude from both author tables and contributor lists
   - Do NOT count excluded authors' PRs in totals

7. **Present results** in markdown tables:

### PRs by Author (merged to main)

| Author    | PRs   | LOC Changed | Avg/PR  | Comments | Approvals |
| --------- | ----- | ----------- | ------- | -------- | --------- |
| @author   | X     | X,XXX       | XXX     | XX       | XX        |
| **Total** | **X** | **X,XXX**   | **XXX** | **XX**   | **XX**    |

### PRs by Repository

| Repo     | PRs | Contributors             |
| -------- | --- | ------------------------ |
| **repo** | XX  | @author (X), @author (X) |

8. **Also provide**:
   - Summary of major themes/initiatives from PR titles
   - Pagination summary: "Fetched X PRs across Y pages for each repository"
   - Link to GitHub insights pages for manual verification:
     - https://github.com/{ORG}/{REPO}/pulse
     - https://github.com/{ORG}/{REPO}/graphs/contributors
