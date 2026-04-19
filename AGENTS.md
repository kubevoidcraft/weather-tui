# AGENTS.md

Concrete conventions for anyone (human or AI) contributing to `weather-tui`.
Prefer rules over principles: each entry below should be something you can
verify with a command or a diff.

## Project at a glance

- **What**: Terminal UI weather client. No API keys, no daemon, no config
  wizard; one binary, two panels, favorites persisted to `~/.config/`.
- **Language/toolchain**: Go (version pinned in `go.mod`; currently `1.25.x`).
- **UI stack**: `github.com/rivo/tview` on top of `github.com/gdamore/tcell/v2`.
- **Data source**: [Open-Meteo](https://open-meteo.com/) — forecast and
  geocoding endpoints, both keyless. Do not introduce an alternate provider
  that requires registration or a token.
- **Targets**: macOS and Linux, `amd64` and `arm64`. Cross-compiled by the
  release workflow.

## Repository layout

```
.
├── main.go                     # entry point
├── ui/                         # tview wiring, rendering, keybindings
│   ├── app.go                  # App struct, setup, event handlers
│   ├── sparkline.go            # sparkline() and shaded() renderers
│   └── *_test.go
├── internal/
│   ├── weather/                # Open-Meteo client + types
│   │   ├── api.go              # SearchCities / GetForecast
│   │   ├── types.go            # City, etc.
│   │   ├── codes.go            # WMO weather codes
│   │   └── *_test.go
│   └── config/                 # persisted preferences (~/.config/weather-tui)
├── .github/
│   ├── workflows/ci.yml        # lint + govulncheck + tests + coverage
│   ├── workflows/release.yml   # auto-tag + cross-compile + GitHub Release
│   └── dependabot.yml          # weekly grouped updates for gomod + actions
├── .golangci.yml               # v2 schema
├── AGENTS.md                   # this file
└── README.md
```

## Working agreement

- **Incremental commits.** One logical change per commit, descriptive subject,
  explain the *why* in the body when it isn't obvious. Never batch unrelated
  changes.
- **Ask before large rewrites.** For anything beyond a self-contained fix,
  surface a plan first and wait for confirmation.
- **Cite sources by file:line.** Reference code as `path/to/file.go:NN` in PR
  descriptions and discussions so reviewers can jump straight there.
- **Don't invent dependencies.** Before adding a module, check: is it
  maintained (commits in the last 12 months)? Does it have known CVEs
  (`govulncheck`)? Is it pulling a larger dependency tree than the problem
  warrants? If any answer is unclear, do not add it.

## Code conventions

### Go style

- Run `gofmt` / `goimports` on every file you touch. CI will reject
  unformatted code via `golangci-lint`.
- Exported identifiers must have doc comments starting with the identifier
  name. Unexported helpers get comments when their purpose isn't obvious
  from the name alone.
- Prefer early returns and small functions. A function that needs a
  paragraph-length comment to explain its control flow is a refactor
  candidate.
- No package-level mutable state except where explicitly justified in a
  comment (e.g. `geomapAPIURL` / `weatherAPIURL` in `internal/weather/api.go`
  are `var` instead of `const` so tests can point them at `httptest` servers;
  production code never mutates them).

### HTTP clients

- Every outbound request must:
  - accept a `context.Context` and use `http.NewRequestWithContext`;
  - send a descriptive `User-Agent` (currently `weather-tui/...`);
  - have a bounded timeout — either on the `http.Client` or the context;
  - cap the response body with `io.LimitReader` (1 MiB for Open-Meteo
    endpoints, which return single-digit KiB in practice).
- Reuse a single `http.Client` per package so connection pooling works.
- Errors returned to the UI layer must be wrapped with `%w` so callers can
  use `errors.Is` / `errors.As`.

### UI layer

- Long-running work (network calls, file IO) runs in a goroutine, and UI
  updates are posted via `App.QueueUpdateDraw`. Never touch tview widgets
  directly from a background goroutine.
- Debounce anything driven by keystroke events that hits the network. The
  search box uses 250 ms (`searchDebounce` in `ui/app.go`) plus in-flight
  cancellation via `context.Context`. Copy that pattern, don't reinvent it.
- Keep emoji / glyph usage minimal. Terminal/font width for variation-selector
  emoji (e.g. `🌡️`) is inconsistent across terminals and cannot be reliably
  fixed at the string level — prefer ASCII or well-supported single-codepoint
  symbols when alignment matters.

### Tests

- Every network-touching function must have an `httptest`-based test. Use
  the `withStubbedURL` helper in `internal/weather/api_test.go` as the
  pattern.
- Tests must pass with `-race`. CI runs `go test -race ./...`; locally you
  should do the same before pushing.
- When adding a feature, add or update tests in the same commit. A feature
  without a test is incomplete.

## Required local checks (before every push)

Run these in order; all must pass:

```bash
go build ./...
go test -race ./...
golangci-lint run ./...
govulncheck ./...          # install once: go install golang.org/x/vuln/cmd/govulncheck@latest
```

If you changed the binary's behavior, also:

```bash
go build -o weather-tui . && ./weather-tui
```

and verify the UI still renders on your terminal. Do not ship UI changes
without a manual smoke test — the automated tests cannot catch rendering
regressions.

## CI and releases

- **`ci.yml`** runs on every push and PR to `main`: lint, `govulncheck`,
  tests with race + coverage. Coverage is uploaded as an artifact. The job
  must be green for `release.yml` to run.
- **`release.yml`** runs on pushes to `main`, auto-increments the patch
  version, tags, and cross-compiles. To skip a release (e.g. docs-only
  changes) include `[skip ci]` in the commit message.
- **`golangci-lint`** is pinned to a specific v2 minor in CI because older
  releases reject the Go version in `go.mod`. If you bump the Go version,
  also verify the pinned `golangci-lint` is compatible.
- **Dependabot** opens weekly grouped PRs for `gomod` and `github-actions`.
  Review and merge promptly; do not let security updates sit.

## Security posture

- Open-Meteo is unauthenticated, so there are no secrets in the repo or
  environment. Keep it that way — do not introduce features that require
  an API key unless there is no alternative, and if you must, use a
  platform-native secret store (never commit `.env` files).
- `govulncheck` is part of CI. If it fails, fix the vulnerability or bump
  the affected module; do not silence the check.
- `go.sum` is committed and must stay consistent with `go.mod`. Use
  `go mod tidy` before committing dependency changes.
- Workflow permissions are minimized (`contents: read`). Don't loosen them
  without a specific reason documented in the diff.

## Documentation

- `README.md` is user-facing: install, run, keybindings. Keep it short and
  current. Any new feature that changes a keybinding or a config file must
  land with a `README.md` update in the same commit.
- `AGENTS.md` (this file) is contributor-facing. Update it when a convention
  changes — not when one is proposed.
- Do not create additional top-level docs (`SKILL.md`, `CONTRIBUTING.md`,
  `ARCHITECTURE.md`, …) without a clear reason. Consolidate into the two
  files above.
