# AGENTS.md

## Expectations
- **Architecture**: Terminal User Interface (TUI) application
- **Platform**: Mac and Linux (arm64, amd64)
- **Deployment**: Automatic releases via `.github/workflows/release.yml` matrices.
- **Language**: Go
- **Workflow**:
  - Work incrementally.
  - Seek user validation on proposed changes.
  - Rely on official documentation (Go docs).
  - Use `golangci-lint` after each change.
- **Dependencies**:
  - Do not use abandoned, unmaintained, or risky libraries.
  - Always check for vulnerabilities and CVEs before finalizing changes.
  - Prefer robust weather APIs that do not require registration/tokens.
- **Code Quality**:
  - Code must be human-readable and clean.
  - Do not introduce overly complex abstractions unless necessary.
- **Documentation**:
  - Maintain a concise `README.md` alongside code development.
