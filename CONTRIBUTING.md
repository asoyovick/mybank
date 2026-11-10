# Contributing to MyBank

Welcome to MyBank. This guide establishes the collaboration standards for the
two-person development team so that feature work stays clean, code quality stays
high, and branches never collide.

> **Core principles**
> - **Clean code, every time.** Formatting is enforced with `gofumpt`; builds and
>   tests must pass before any pull request is merged.
> - **Independent feature development.** Each task gets its own branch. Nobody
>   works directly on `main`.
> - **Direct communication between partners.** If a task is ambiguous, blocked,
>   or exceeds a few hours of work, raise it now — not after the PR is open.

---

## 1. Overview & Collaboration Rules

### 1.1 Code Quality Standards

- **Formatting:** Every Go file must be formatted with `gofumpt`.
  ```bash
  gofumpt -w .
  ```
- **Build:** `go build ./...` must complete with zero errors.
- **Vet:** `go vet ./...` must produce no findings.
- **Structure:** Keep the layered layout intact — `cmd/` for the entry point,
  `internal/model/` for domain structs, `internal/bank/` for business rules,
  `internal/database/` for PostgreSQL persistence.

### 1.2 Branch Independence

- Never commit directly to `main`.
- Never push to `main` from a local machine.
- A PR may only merge when both partners agree it meets the checklist below.

### 1.3 Communication

- Open a **GitHub Discussion** or **issue** before starting a task this large.
- Use **standup chats** to pass context between partners at the start of each
  day. Include: what you finished, what you are doing, and what is blocking you.
- Keep comments in the PR short and actionable: point to the behavior, not just
  the line.

---

## 2. Branching Strategy & Naming Conventions

All work happens on topic branches. The `main` branch is the integration
branch and must always be deployable.

### 2.1 `main` Branch Protection

`main` is the source of truth and the release branch. Protect it with:

- **No direct pushes** — all changes come through a PR.
- **Require at least one approving review** from the other partner before merge.
- **Require CI to pass** (build, vet, and any test suites) before merge.
- **Require up-to-date branch** — rebase or merge `main` into the topic branch
  before merging if `main` has moved.

### 2.2 Branch Naming

Use the exact prefixes below. Do not use spaces or special characters.

| Type | Prefix | Example | When to use |
|---|---|---|---|
| Feature | `feature/` | `feature/postgres-integration` | New functionality |
| Bug fix | `fix/` | `fix/balance-rounding` | Correcting a defect |
| Documentation | `docs/` | `docs/update-readme` | Docs, README, or `CONTRIBUTING` changes |

**Examples:**

```bash
git checkout -b feature/postgres-integration
git checkout -b fix/balance-rounding
git checkout -b docs/update-readme
```

---

## 3. Step-by-Step Git Workflow

Here is the full life of a task, from starting to merging.

### Step 1: Pull the Latest `main`

```bash
git checkout main
git pull origin main
```

Always start from the latest `main` so your branch has the most recent context.
Before creating a branch, confirm no one else has already claimed a similar
name via `git branch -r` or the GitHub branch list.

### Step 2: Create a Dedicated Topic Branch

```bash
git checkout -b feature/postgres-integration
```

- One topic branch per task. Never mix multiple features in one branch.
- Use the naming prefix from Section 2.
- Keep branches small and mergeable quickly.

### Step 3: Implement the Feature

- Keep commits **atomic** and **descriptive**.
- Commit as you go, then clean up with interactive rebase before opening the PR.

```bash
git add internal/database/db.go
git commit -m "Add PostgreSQL connection pool to internal/database"

git add cmd/main.go internal/bank/bank.go
git commit -m "Wire app startup to open the DB pool and pass it to bank.NewBank"
```

**Commit message rules:**

- Use the imperative mood: `Add ...`, `Fix ...`, `Remove ...`.
- Keep the subject under ~72 characters.
- Add a body only when the change is non-obvious.

### Step 4: Push the Branch

```bash
git push origin feature/postgres-integration
```

- If the remote rejects a force-push because the branch already exists, rebase
  on `main` first and then `git push --force-with-lease`.
- Never force-push someone else's branch.

### Step 5: Open a Pull Request

1. Push the branch.
2. Open the PR on GitHub with a clear title and description.
3. Describe **what** changed and **why**.
4. Request a review from the other partner.
5. Run the **Quality Checklist** (Section 4) locally before requesting review.
6. Address review comments, keep commits tidy, and note updates in the PR.
7. Once both partners approve and CI passes, merge (squash or rebase merge, per
   team preference) and delete the branch.

**Merge checklist:**

- [ ] PR is up to date with `main`.
- [ ] `gofumpt -w .` ran with no files left unformatted.
- [ ] `go build ./...` passes.
- [ ] `go vet ./...` is clean.
- [ ] Reviewers approve.
- [ ] Branch is deleted after merge.

---

## 4. Code Review & Quality Checklist

Before a PR is merged, the following must be true. This is a **gate**, not a
suggestion.

| Check | What to verify | How to verify |
|---|---|---|
| **Compiles** | Zero build errors across all packages. | `go build ./...` |
| **Vet clean** | No static-analysis findings. | `go vet ./...` |
| **Formatted** | `gofumpt` changes nothing. | `gofumpt -d .` prints nothing. |
| **Package integrity** | Layering is respected; `internal/` boundaries are not crossed. | Visual review of imports. |
| **No sensitive data** | No secrets, tokens, or credentials are committed. | `git log -p --all | grep -iE 'pass|secret|key|token'` |
| **Clear comments** | Non-obvious logic, SQL, and financial calculations are documented. | Read the code. |
| **Test coverage** | New behavior has a test, or a justification for no test. | Look for `_test.go` or grep for `Test`. |
| **Database correctness** | Money uses `NUMERIC(12,2)`, queries are parameterized, transactions wrap multi-step writes. | Review schema + SQL. |

**Pre-merge minimum commands:**

```bash
gofumpt -w .
go build ./...
go vet ./...
go test ./... 2>/dev/null || echo "no tests configured; CI will block merge"
```

**After merging**, delete the remote branch:

```bash
git push origin --delete feature/postgres-integration
```

---

## 5. Communication Norms

- **Blockers** are raised in the PR or a dedicated issue, not in chat only.
- **Breakdowns** are surfaced the same day and result in a quick pair call.
- **Review turnaround** is expected within one business day whenever possible.
- **Disagreements** are resolved by the evidence in the PR and the shared goal:
  a shipping, correct, and maintainable banking system.

---

## Appendix A — Common Commands

```bash
# Pull the latest main
git checkout main && git pull origin main

# Create a topic branch
git checkout -b feature/postgres-integration

# Stage and commit
git add <files>
git commit -m "Add ..."

# Push
git push origin feature/postgres-integration

# Rebase on main when needed
git fetch origin
git rebase origin/main

# Merge PR
git checkout main
git pull origin main

# Delete a remote branch
git push origin --delete feature/postgres-integration
```

---

## Appendix B — Quick Reference

```text
main       -> protected integration branch. Never commit directly.
feature/   -> new functionality
fix/       -> bug fixes
docs/      -> documentation changes
```

---

*Maintained by the MyBank development team. Suggestions for this guide are
welcome as PRs to `docs/CONTRIBUTING.md`.*
