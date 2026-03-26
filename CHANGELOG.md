# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-03-26

First stable release of **depscount**: a multi-signal dependency risk scanner for **npm** and **PyPI** projects, with policy-as-code and CI-friendly exit codes.

### Added

- **CLI** (`depscount`) built with Cobra:
  - `scan [path]` — run a scan and print a terminal summary (default path: current directory).
  - `ci [path]` — same pipeline with exit codes for automation (see below).
- **Global flags**: `--config`, `--offline`, `--json`, `--no-color`, `--quiet`, `--verbose`, `--cache-dir` (default cache path only; on-disk caching not yet wired through the scanner).
- **Project discovery** — walks upward from the scan path for npm (`package.json`) or Python (`pyproject.toml`, `requirements.txt`, `Pipfile`).
- **Resolution** — lockfile-first graphs:
  - npm: `package-lock.json` (v1–3 style handling).
  - PyPI: `Pipfile.lock`; pinned direct deps from `requirements.txt` / `pyproject` when no lockfile.
- **Registry enrichment** (skipped with `--offline`) — npm Registry and PyPI JSON APIs for license strings and publish activity timestamps.
- **Risk engines** (run concurrently):
  - **Security** — [OSV](https://osv.dev/) `querybatch` API for known vulnerabilities per concrete package version.
  - **Abandonment** — score from registry activity age (no GitHub API in this release).
  - **License** — SPDX-style tiering from declared license metadata plus policy rules.
- **Composite scoring** — configurable dimension weights in YAML; depth multipliers so deep transitive dependencies weigh less than direct ones. Supply-chain weight in config is folded into other dimensions until a dedicated engine ships.
- **Policy** (`.depscount.yml` / `depscount.yml`) — `weights`, `thresholds.fail_on`, `licenses.allowed`, `licenses.blocked`, `treat_unknown_as` (`warn` | `fail` | `ignore`).
- **Legacy config filenames** — `.depguard.yml` / `depguard.yml` still discovered after the new names.
- **Output** — lipgloss-styled terminal report and full structured **`ScanReport` JSON** with `--json`.
- **Environment variables** — `DEPSCOUNT_CACHE_DIR`, `DEPSCOUNT_OFFLINE`, `DEPSCOUNT_NO_COLOR`; legacy `DEPGUARD_*` aliases for cache, offline, and no-color.
- **Documentation** — `README.md`, `CONTRIBUTING.md`, and GitHub Actions workflow example (`.github/workflows/depscount.yml`).

### CI exit codes (`depscount ci`)

| Code | Meaning |
|------|---------|
| 0 | Clean relative to `fail_on` and no blocker policy violations |
| 1 | Tool or scan setup error |
| 2 | Risk threshold breached and/or non-blocker policy violations |
| 3 | Blocker policy violations (e.g. blocked license, `treat_unknown_as: fail`) |

### Security & privacy

- Scans send **package names and versions** (and receive public registry / OSV data only). Application source code is not uploaded.
- **`--offline`** disables OSV and registry enrichment.

### Known limitations (v1.0.0)

- No persistent **SQLite** advisory/registry cache yet (repeat scans still hit the network).
- No **supply-chain** engine (typosquatting, dependency confusion, etc.) — only a reserved weight in config.
- No **SARIF**, **`diff`**, **`explain`**, **`update-db`**, or **`policy validate`** subcommands (see architecture roadmap).
- **`--cache-dir`** is not yet connected to scanner persistence.
- **Config discovery** walks from the **current working directory**, not automatically from the scanned path; use `cd` into the project or `--config` when needed.

### Install

```bash
go install github.com/depscount/depscount/cmd/depscount@v1.0.0
```

Or build from source: `make build` → `./bin/depscount`.

---

[1.0.0]: https://github.com/depscount/depscount/releases/tag/v1.0.0
