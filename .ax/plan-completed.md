# Completed Work

<!-- See also: [Plan](plan.md) | [Roadmap](plan-roadmap.md) -->

## v0.3.17 (2026-04-10)

### Slot 10: refactor: error handling, login cleanup, and test coverage (PR #260)
- [x] #255 — refactor: consolidate scattered timeout constants
- [x] #249 — refactor: wrap bare error returns in initLocked and closeBrowser
- [x] #250 — refactor: stop ignoring marshal and page.Info errors
- [x] #251 — refactor: add debug logging to walkIframeNode silent failures
- [x] #253 — refactor: extract walkScalarNode into focused helper methods
- [x] #246 — refactor: break LoginHandler into focused helpers
- [x] #247 — refactor: deduplicate login credential-filling between trigger and standard paths
- [x] #254 — refactor: make login selector defaults configurable
- [x] #252 — refactor: add unit tests for login form detection and filling

### fix: rod_login fail fast on HTTP errors (PR #257)
- [x] #256 — fix: rod_login should fail fast on non-200 HTTP status

## v0.3.16 and earlier

### Slot 9: feat: auth workflows (~5h)
- [x] #207 — feat: rod_login — configurable auth flow in one call
- [x] #208 — feat: cookie persistence — save/restore across MCP restarts

### Slot 8: feat: batch + error context (~4h)
- [x] #201 — feat: rod_batch — execute multiple actions in one MCP call
- [x] #205 — feat: rod_compare_screenshots — visual diff between two images

### Slot 7: feat: mobile + auth emulation (~4h)
- [x] #228 — feat: geolocation emulation
- [x] #229 — feat: touch/tap events for mobile emulation
- [x] #231 — feat: HTTP basic auth handling

### Slot 6: feat: downloads + race selector (~4h)
- [x] #227 — feat: file download handling
- [x] #224 — feat: race selector for branching page outcomes

### Slot 5: feat: frame + shadow DOM access (~6h)
- [x] #226 — feat: iframe and frame navigation support
- [x] #223 — feat: shadow DOM access for modern web components

### Slot 4: feat: compound tools (~5h)
- [x] #200 — feat: rod_navigate_and_snapshot — combined navigate + wait + snapshot
- [x] #203 — feat: rod_wait_for_text — wait for specific text to appear on page
- [x] #202 — feat: rod_assert_element — assert existence + screenshot in one call

### Slot 3: feat: click + keyboard enhancements (~4h)
- [x] #221 — feat: right-click and double-click support in rod_click
- [x] #225 — feat: keyboard modifier combos (Ctrl+A, Ctrl+C, Shift+Click)

### Slot 2: feat: page info + status (~3h)
- [x] #222 — feat: rod_page_info tool for current URL and title
- [x] #209 — feat: rod_page_status — quick page health check in one call
- [x] #206 — feat: include page context in error responses

### Slot 1: feat: screenshot overhaul (~5h)
- [x] #220 — feat: full-page screenshot support
- [x] #204 — feat: rod_screenshot improvements — selector, save_to, viewport default
- [x] #230 — feat: element screenshot for component-level captures

### Slot 0: fix: login, fill, and snapshot reliability (~6h)
- [x] #243 — bug: snapshot refs invalidated between consecutive tool calls
- [x] #242 — bug: react_clipboard fill doesn't trigger React state update for auth forms
- [x] #241 — feat: rod_login support for modal-based login flows

## v0.3.6 (2026-03-29)

- [x] #217 — feat: redact password field values in ARIA snapshots (PR #219)

## Post v0.3.5 (2026-03-29)

- [x] #211 — docs: add AGENTS.md for AI agent guidelines (PR #218)

## v0.3.5 (2026-03-29)

- [x] #214 — fix: rod_click fails with "no snapshot available" after rod_fill_form (PR #216)
- [x] #212 — fix: snapshot from rod_fill/rod_click not persisted for subsequent ref-based ops (PR #216)
- [x] #215 — feat: add submit option to rod_fill_form for one-call form submission (PR #216)

## v0.3.4 (2026-03-28)

- [x] #194 — feat: support CSS selectors in rod_click and rod_fill (PR #213)
- [x] #197 — feat: smart fill for React controlled inputs (PR #213)
- [x] #199 — feat: rod_fill_form — batch fill multiple fields (PR #213)

## v0.3.3 (2026-03-28)

- [x] #195 — fix: rod_evaluate returns nil for string values (PR #210)
- [x] #196 — fix: aria snapshot crashes on React SPAs (PR #210)
- [x] #198 — fix: browser connection dies after ~15min inactivity (PR #210)
- [x] #193 — feat: add rod_type tool for typing strings (PR #210)

## v0.3.2 (2026-03-24)

- [x] #187 — fix: auto-accept beforeunload dialogs and add navigation timeout
- [x] #188 — refactor: completely detach from upstream go-rod/rod-mcp
- [x] #191 — feat: add auto-merge workflow
