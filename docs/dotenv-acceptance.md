# Dotenv vertical — acceptance criteria

Function-level acceptance criteria for the `.env` commands (`doctor`, `hydrate`,
`make`) and their shared plumbing. These are finer-grained than the product ACs
in [user-stories.md](./user-stories.md) (Epic C, L1, A2) — they map to code and
each is pinned to the test that proves it. Update this table when behavior changes.

## `MainWorktree(cwd)` — `internal/check/worktree.go`

Finds the repo's primary worktree (first `git worktree list --porcelain` entry).

| # | Criterion | Test |
|---|-----------|------|
| 1 | From a **linked** worktree's dir → returns the **main** worktree's path | `TestMainWorktree` |
| 2 | From the main worktree itself → returns the main worktree path | `TestMainWorktree` |
| 3 | Outside any git repo → non-nil error, no panic, no bogus path | `TestMainWorktree` |
| 4 | Returned path exists and is a directory | `TestMainWorktree` |

## `Worktree.EnvVarsByDir()` — `internal/check/worktree.go`

Indexes `.env` files by directory relative to the worktree root (the shared
primitive under doctor's fallback, hydrate's fill, and make's generation).

| # | Criterion | Test |
|---|-----------|------|
| 1 | Indexes only `.env`, never `.env.example` (example-only dir → no entry) | `TestEnvVarsByDir` |
| 2 | Keys are dirs relative to root (`svc_a`; `.` for root-level `.env`) | `TestEnvVarsByDir` |
| 3 | Value map equals the parsed vars (keys **and** values) | `TestEnvVarsByDir` |
| 4 | No `.env` files → empty, non-nil map | `TestEnvVarsByDirEmpty` |

## `doctor` reference resolution — `internal/check/doctor.go` (`CheckEnv`)

Expected keys come from a service's `.env.example`, else the main worktree's `.env`.

| # | Criterion | Test |
|---|-----------|------|
| 1 | With `.env.example` present, it is the reference (missing/empty keys reported) | `TestCheckEnv` |
| 2 | No source worktree → `.env.example`-only behavior; a `.env` with no example is skipped | `TestCheckEnv` |
| 3 | No `.env.example` → falls back to main's `.env` for the key set | `TestCheckEnvMainFallback` |
| 4 | Service present in main but `.env` absent here → flagged `NoEnv` | `TestCheckEnvMainFallback` |

## `hydrate` planning — `internal/check/hydrate.go` (`PlanHydrate`)

Plans append-only repairs for **missing** keys, valued from the source worktree.

| # | Criterion | Test |
|---|-----------|------|
| 1 | Missing key present in source → planned with the source's value | `TestPlanHydrate` |
| 2 | Missing key absent from source → planned empty, listed as `Unsourced` | `TestPlanHydrate` |
| 3 | Healthy service → no repair | `TestPlanHydrate` |
| 4 | `.env` absent entirely → `Create` repair (vs append) | `TestPlanHydrate` |
| 5 | Source lacks the service dir → every key `Unsourced` | `TestPlanHydrateNoSourceService` |

## `make` command — `cmd/make.go`

Generates `.env.example` from each service's `.env`, keys copied, values blanked.

| # | Criterion | Test |
|---|-----------|------|
| 1 | Each service `.env` → sibling `.env.example`, same keys, **values blanked** (no secret leak) | `TestMake/generate_blanks_values` |
| 2 | Existing `.env.example` is never overwritten (skip reported) | `TestMake/skip_existing_example` |
| 3 | Empty worktree → falls back to main's `.env` files, generates from those | `TestMake/fallback_to_main_worktree` |
| 4 | No `.env` anywhere → honest "no .env files found" message, creates nothing | `TestMake/nothing_anywhere` |

---

Deliberate non-goals (documented in code): present-but-empty keys are doctor's
nag but not hydrated (v2, needs in-place edits); `make` is create-only and won't
sync new keys into an existing `.env.example` (v2 `--sync`).
