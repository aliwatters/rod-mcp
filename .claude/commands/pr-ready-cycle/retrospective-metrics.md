# PR Ready Retrospective Metrics Collection

# ⚠️ SILENT MODE - UNATTENDED EXECUTION ⚠️

**CRITICAL**: This workflow runs UNATTENDED without user intervention.

**Agent Requirements:**

1. Execute ALL commands immediately without pausing
2. Execute `prompt` transitions immediately
3. Do NOT provide status summaries
4. Do NOT wait for user confirmation
5. Do NOT stop between phases
6. Let command output speak for itself
7. Only stop on errors or completion

**Expected User Intervention**: ZERO (after initial command)

---

**Purpose**: Collect and verify session metrics for retrospective analysis.

## PR Label Success Verification

**Purpose**: Verify that PRs actually reached intended "ready for humans" + "verified" state.

```bash
# Get list of processed PRs
PRS_PROCESSED=$(ax prready get | grep "^PRS_PROCESSED=" | cut -d= -f2)

echo "🔍 Verifying label states for processed PRs..."
SUCCESS_COUNT=0
PARTIAL_COUNT=0
FAILED_COUNT=0

# Check each PR
# Use tr to convert commas to newlines, handling spaces around commas
for PR in $(echo "$PRS_PROCESSED" | tr ',' '\n' | tr -d ' '); do
    if [ -z "$PR" ] || [ "$PR" = "0" ]; then
        continue
    fi

    echo "   Checking PR #$PR..."

    # Get labels for this PR
    # Exit code handling: if ax pr view fails, LABELS will be empty string
    LABELS=$(ax pr view "$PR" --json labels --jq '.labels[].name' 2>/dev/null | tr '\n' ',')
    if [ $? -ne 0 ]; then
        LABELS=""
    fi

    # Check for success states using exact word matching
    if echo "$LABELS" | grep -qw "ready for humans" && echo "$LABELS" | grep -qw "verified"; then
        echo "   ✅ PR #$PR: Successfully verified (ready for humans + verified)"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    elif echo "$LABELS" | grep -qw "ready for humans"; then
        echo "   ⚠️  PR #$PR: Ready for humans but NOT verified"
        echo "      This indicates verification phase was skipped or failed"
        PARTIAL_COUNT=$((PARTIAL_COUNT + 1))

        # Log for retrospective analysis
        mkdir -p tmp
        echo "## PR #$PR: Missing Verified Label" >> tmp/friction-pr-ready-session.md
        echo "- Has 'ready for humans' label" >> tmp/friction-pr-ready-session.md
        echo "- Missing 'verified' label" >> tmp/friction-pr-ready-session.md
        echo "- Possible causes: verification phase skipped, validation failed, transition bug" >> tmp/friction-pr-ready-session.md
        echo "" >> tmp/friction-pr-ready-session.md
    elif echo "$LABELS" | grep -qw "needs-attention"; then
        echo "   ⚠️  PR #$PR: Needs attention (verification failed)"
        FAILED_COUNT=$((FAILED_COUNT + 1))
    else
        echo "   ❌ PR #$PR: No ready labels (workflow may have failed)"
        FAILED_COUNT=$((FAILED_COUNT + 1))

        # Log for retrospective analysis
        mkdir -p tmp
        echo "## PR #$PR: No Ready Labels" >> tmp/friction-pr-ready-session.md
        echo "- No 'ready for humans' or 'ready for robots' label" >> tmp/friction-pr-ready-session.md
        echo "- Workflow may have failed or been interrupted" >> tmp/friction-pr-ready-session.md
        echo "" >> tmp/friction-pr-ready-session.md
    fi
done

echo ""
echo "📊 Label Success Summary:"
echo "   ✅ Verified: $SUCCESS_COUNT"
echo "   ⚠️  Partial: $PARTIAL_COUNT (has ready label but not verified)"
echo "   ❌ Failed: $FAILED_COUNT (no ready labels or blocked)"

# Update metrics
ax prready set-metric verified_success_count "$SUCCESS_COUNT"
ax prready set-metric partial_success_count "$PARTIAL_COUNT"
ax prready set-metric label_failure_count "$FAILED_COUNT"

# Create issue for unexpected failures (partial success indicates likely bug)
if [ "$PARTIAL_COUNT" -gt 0 ]; then
    echo ""
    echo "⚠️  Warning: $PARTIAL_COUNT PR(s) have 'ready for humans' but NOT 'verified'"
    echo "   This indicates a potential workflow bug (verification phase skipped)"
    echo "   Consider investigating transition logic in prepare-for-verified phase"
fi
```

## Session Performance Metrics

```bash
# Get detailed metrics
CONFLICTS_RESOLVED=$(ax prready get | grep "^CONFLICTS_RESOLVED=" | cut -d= -f2)
LINT_FIXES=$(ax prready get | grep "^LINT_FIXES=" | cut -d= -f2)
TEST_FAILURES=$(ax prready get | grep "^TEST_FAILURES=" | cut -d= -f2)
COMPLEX_ISSUES=$(ax prready get | grep "^COMPLEX_ISSUES=" | cut -d= -f2)
TOTAL_TIME=$(ax prready get | grep "^TOTAL_PREP_TIME_MINUTES=" | cut -d= -f2)

echo "📈 Performance Analysis:"
echo "   Conflicts resolved: $CONFLICTS_RESOLVED"
echo "   Lint fixes applied: $LINT_FIXES"
echo "   Test failures: $TEST_FAILURES"
echo "   Complex issues: $COMPLEX_ISSUES"
echo "   Total time: ${TOTAL_TIME} minutes"
```

## Expected Output

### Successful Session

```text
🔍 Verifying label states for processed PRs...
   Checking PR #1234...
   ✅ PR #1234: Successfully verified (ready for humans + verified)
   Checking PR #1235...
   ✅ PR #1235: Successfully verified (ready for humans + verified)
   Checking PR #1236...
   ✅ PR #1236: Successfully verified (ready for humans + verified)

📊 Label Success Summary:
   ✅ Verified: 3
   ⚠️  Partial: 0 (has ready label but not verified)
   ❌ Failed: 0 (no ready labels or blocked)

📈 Performance Analysis:
   Conflicts resolved: 5
   Lint fixes applied: 12
   Test failures: 1
   Complex issues: 0
   Total time: 45 minutes
```

### Session with Issues

```text
🔍 Verifying label states for processed PRs...
   Checking PR #1237...
   ⚠️  PR #1237: Ready for humans but NOT verified
      This indicates verification phase was skipped or failed
   Checking PR #1238...
   ❌ PR #1238: No ready labels (workflow may have failed)

📊 Label Success Summary:
   ✅ Verified: 0
   ⚠️  Partial: 1 (has ready label but not verified)
   ❌ Failed: 1 (no ready labels or blocked)

⚠️  Warning: 1 PR(s) have 'ready for humans' but NOT 'verified'
   This indicates a potential workflow bug (verification phase skipped)
   Consider investigating transition logic in prepare-for-verified phase

📈 Performance Analysis:
   Conflicts resolved: 1
   Lint fixes applied: 3
   Test failures: 2
   Complex issues: 1
   Total time: 30 minutes
```

## Troubleshooting

### Metrics Missing

**Cause**: State file corrupted or incomplete

**Solution**:

```bash
# Display raw state
ax prready get

# Use defaults if missing
PR_COUNT=${PR_COUNT:-0}
COMPLEX_ISSUES=${COMPLEX_ISSUES:-0}
```

### Label Verification Fails

**Cause**: GitHub API rate limits or network issues

**Solution**:

```bash
# Retry with exponential backoff
for attempt in 1 2 3; do
    if ax pr view "$PR" --json labels --jq '.labels[].name' 2>/dev/null; then
        break
    else
        echo "   Attempt $attempt failed, retrying in $((attempt * 2)) seconds..."
        sleep $((attempt * 2))
    fi
done
```
