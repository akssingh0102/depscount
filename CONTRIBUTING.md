# Contributing to depscount

Thank you for taking the time to contribute. This document explains how we work on the project and what we look for in changes.

## Ground rules

- **Be respectful.** Assume good intent, keep feedback specific and actionable.
- **Prefer small, focused PRs.** One logical change per pull request is easier to review and revert if needed.
- **Match existing style.** Naming, package layout, and error handling should feel consistent with the surrounding code.
- **Do not break the CLI contract without discussion.** Flags, exit codes, and JSON report shape are part of the public API for integrators.

## Getting started

### Prerequisites

- **Go 1.22 or newer** (`go version`)

### Clone and build

```bash
git clone https://github.com/depscount/depscount.git
cd depscount
go mod tidy
make build
```

Run the binary:

```bash
./bin/depscount scan --help
./bin/depscount ci --help
```

### Run tests

```bash
make test
# or
go test ./...
```

When adding behavior that touches HTTP or registries, prefer **table-driven unit tests** and, where practical, **hermetic tests** (fixtures, `httptest`, or small golden files) so CI does not depend on live APIs.

### Sanity checks before you push

```bash
go vet ./...
go fmt ./...
```

Optional but appreciated:

```bash
golangci-lint run   # if you use golangci-lint locally
```

## What to contribute

High-value areas:

- **Correctness** in manifest parsing, lockfile handling, and graph resolution  
- **Engine** improvements (security, abandonment, license) with clear scoring rationale  
- **Policy** behavior and documentation when config semantics change  
- **Tests** and **fixtures** for edge cases (monorepos, scoped npm packages, PEP 440 versions, etc.)  
- **Docs** (README, this file) when behavior or flags change  

Please open an issue first for **large features** (new ecosystem, SARIF, persistent cache, new subcommands) so design and scope can be agreed on.

## Pull request checklist

- [ ] `go test ./...` passes  
- [ ] `go vet ./...` is clean  
- [ ] Code is `gofmt`-formatted  
- [ ] User-visible behavior is documented in **README.md** (or this file) when it changes  
- [ ] JSON output or exit codes: call out any intentional change in the PR description  

## Commit messages

Use clear, imperative summaries, for example:

- `fix(parser): handle missing lockfile devDependencies`  
- `feat(security): chunk OSV batch requests under limit`  
- `docs: clarify ci exit codes`  

The optional `(scope)` prefix is welcome when it aids history search.

## Project layout (orientation)

| Path | Role |
|------|------|
| `cmd/depscount/` | CLI entrypoint |
| `internal/cli/` | Cobra commands and flags |
| `internal/scanner/` | Scan orchestration |
| `internal/parser/` | Manifest / lockfile parsing |
| `internal/resolver/` | Dependency graph |
| `internal/regmeta/` | Registry metadata fetch |
| `internal/engine/` | Risk engines |
| `internal/scorer/` | Composite scoring and report assembly |
| `internal/policy/` | `.depscount.yml` loading and evaluation |
| `internal/output/` | Terminal and JSON formatters |
| `pkg/types/` | Shared domain types |

Deeper design intent lives in the repository’s **dependency risk scanner architecture** document (if present in your tree).

## Security issues

If you believe you have found a security vulnerability in depscount, please **do not** open a public issue with exploit details. Instead, contact the maintainers privately (use GitHub Security Advisories if enabled for the repository, or the contact method the maintainers publish in the README).

## License

By contributing, you agree that your contributions will be licensed under the same terms as the project. **Add or clarify a root `LICENSE` file** in the repository if one is not yet present, so contributors know the exact license.

---

Thank you again for helping improve depscount.
