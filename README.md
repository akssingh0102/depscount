# depscount

**depscount** is a dependency risk scanner for software projects. It resolves your dependency graph (with lockfile-first accuracy), enriches packages from public registries, and scores each dependency across multiple dimensions—not only security vulnerabilities.

| Dimension | What it considers |
|-----------|-------------------|
| **Security** | Known vulnerabilities via [OSV](https://osv.dev/) (batch API) |
| **Abandonment** | Registry publish / activity signals (no GitHub API in current release) |
| **License** | SPDX-style classification from registry metadata + policy allow/block lists |
| **Supply chain** | Reserved weight in config; dedicated engine is on the roadmap |

Output is designed for **local development** (readable terminal summary), **automation** (`--json`), and **CI gates** (`ci` subcommand with stable exit codes).

---

## Why use this?

Many tools optimize for CVE volume. depscount treats **security as one signal among several**, applies **configurable weights**, discounts **deep transitive** dependencies, and surfaces **policy violations** (for example, blocked licenses) as first-class results.

---

## Requirements

- **Go 1.22+** (to build from source)

---

## Install

### Build from the repository

```bash
git clone https://github.com/depscount/depscount.git
cd depscount
go mod tidy
make build
```

The binary is written to **`bin/depscount`**. Add it to your `PATH`, or invoke it by full path.

### Install with `go install`

From a clone or any module path that resolves to this module:

```bash
go install github.com/depscount/depscount/cmd/depscount@latest
```

(Ensure `$GOPATH/bin` or `$(go env GOPATH)/bin` is on your `PATH`.)

---

## Quick start

```bash
# Scan the current directory (discovers npm or Python projects)
depscount scan

# JSON for scripts and dashboards
depscount scan --json > report.json

# CI gate (non-zero exit on policy / threshold breaches)
depscount ci
```

Point at a specific tree:

```bash
depscount scan /path/to/project
depscount ci /path/to/project
```

---

## Commands

| Command | Purpose |
|---------|---------|
| `depscount scan [path]` | Run a scan and print results (default: current working directory). |
| `depscount ci [path]` | Same scan, then exit with a CI-oriented status code (see below). |

Global flags:

| Flag | Description |
|------|-------------|
| `--config <file>` | Path to policy file. If omitted, depscount walks upward from the **current working directory** looking for a config file. |
| `--offline` | Skip network I/O (no OSV queries, no registry enrichment). |
| `--json` | Emit the full `ScanReport` as JSON on stdout. |
| `--no-color` | Disable ANSI colors in terminal output. |
| `--quiet` | Suppress non-essential terminal output. |
| `--verbose` | Reserved for extended logging (hooked for future use). |
| `--cache-dir <dir>` | Default cache location for future on-disk caching; persistence is not yet wired through the scanner. |

---

## Configuration

### Discovery order

depscount searches upward from the **process working directory** (not automatically from the scanned path) for, in order:

1. `.depscount.yml` or `depscount.yml`
2. **Legacy:** `.depguard.yml` or `depguard.yml`

To scan another directory while using a policy file next to that project, run:

```bash
cd /path/to/project && depscount scan .
```

or pass an explicit `--config`.

### Example `.depscount.yml`

```yaml
version: 1

project:
  name: my-service

weights:
  security: 0.45
  abandonment: 0.25
  license: 0.20
  supply_chain: 0.10   # weight is folded into other dimensions until supply-chain engine ships

thresholds:
  fail_on: high        # none | any | low | medium | high | critical | blocker

licenses:
  allowed:
    - MIT
    - Apache-2.0
    - BSD-2-Clause
    - BSD-3-Clause
    - ISC
    - CC0-1.0
  blocked:
    - AGPL-3.0
  treat_unknown_as: warn   # warn | fail | ignore
```

**Policy behavior (summary):**

- **`licenses.blocked`** — blocker violation (CI exit **3**).
- **`licenses.allowed`** — if the allow list is non-empty, licenses not listed generate a **non-blocker** policy hit (CI exit **2** when combined with `ci` logic).
- **`treat_unknown_as: fail`** — missing or unknown license IDs become **blocker** violations.

---

## How a scan works

1. **Discover** an npm or Python project root (walking up from the scan path).
2. **Parse** manifests (`package.json`, `pyproject.toml`, `requirements.txt`, `Pipfile`).
3. **Resolve** dependencies, **preferring lockfiles** (`package-lock.json`, `Pipfile.lock`) for a full, pinned graph.
4. **Enrich** each package from the **npm** or **PyPI** registry (license, publish timestamps)—skipped in `--offline` mode.
5. **Run engines** concurrently:
   - **Security:** OSV batch query per concrete version (semver-looking ranges are skipped).
   - **Abandonment:** score from time since registry activity.
   - **License:** tiered risk from normalized license strings.
6. **Score** a composite per package using configured weights and **depth multipliers** (direct deps weigh more than deep transitives).
7. **Evaluate policy** against the scored report.
8. **Render** terminal and/or JSON output.

---

## Supported ecosystems (current)

| Ecosystem | Manifest | Lockfile (preferred) |
|-----------|----------|----------------------|
| **npm** | `package.json` | `package-lock.json` |
| **PyPI** | `pyproject.toml`, `requirements.txt`, `Pipfile` | `Pipfile.lock` |

Without a lockfile, resolution is **best-effort** (direct dependencies and unpinned ranges may limit OSV accuracy).

---

## CI integration

### Exit codes (`depscount ci`)

| Code | Meaning |
|------|---------|
| `0` | No blocker policy violations and no package severity at or above `thresholds.fail_on`. |
| `1` | Tool or scan setup error (invalid config, no manifest, parse failure, etc.). |
| `2` | Risk threshold breached **or** a **non-blocker** policy violation (e.g., license not in allow list). |
| `3` | **Blocker** policy violation (e.g., blocked license, `treat_unknown_as: fail`). |

### GitHub Actions (example)

The repository includes **`.github/workflows/depscount.yml`**, which checks out the code, builds the binary with Go 1.22, and runs `depscount ci .` on the default branch and pull requests.

---

## Environment variables

| Variable | Purpose |
|----------|---------|
| `DEPSCOUNT_CACHE_DIR` | Intended cache root (default: `~/.depscount/cache`). |
| `DEPSCOUNT_OFFLINE` | If `1` / `true` / `yes`, same as `--offline`. |
| `DEPSCOUNT_NO_COLOR` | If `1` / `true` / `yes`, same as `--no-color`. |

For backward compatibility, `DEPGUARD_CACHE_DIR`, `DEPGUARD_OFFLINE`, and `DEPGUARD_NO_COLOR` are also honored where noted above.

---

## Privacy and network use

- **No application source code** is uploaded. Requests use **package names and versions** (and registry metadata responses are standard public JSON).
- **`--offline`** avoids OSV and registry calls; results will reflect reduced metadata and no live vulnerability data.

---

## Roadmap (high level)

- Persistent **SQLite** cache for advisories and registry responses  
- **Supply-chain** engine (typosquatting, metadata integrity, depth signals)  
- Additional ecosystems (**Go**, **Rust**, **Maven**, …) per design doc  
- **SARIF**, `diff`, `explain`, and richer upgrade suggestions  

See the project’s **dependency risk scanner architecture** document for the full technical design (filename may vary by checkout).

---

## Development

```bash
make tidy    # go mod tidy
make build   # ./bin/depscount
make test    # go test ./...
```

---

## Contributing

Issues and pull requests are welcome. Please keep changes focused and aligned with the architecture document when touching scanner behavior or public CLI contracts.

---

## License

This repository does not yet include a `LICENSE` file. Add one to clarify distribution terms before publishing releases.

## Acknowledgments

Vulnerability data is powered by the **[OSV](https://osv.dev/)** project and API.
