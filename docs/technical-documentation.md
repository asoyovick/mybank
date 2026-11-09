# MyBank — Technical Documentation

> **Audience:** Software engineers, backend students, and systems learners who want to go from “Hello, world” to the structural and safety concerns of a real financial backend.
> **Target architecture:** Go (standard library) + PostgreSQL, layered clean architecture at the source level. This document describes the design as specified and as the codebase is positioned to implement it.

---

## 1. Project Overview & Learning Objectives

### 1.1 What This Project Is

MyBank is a command-line banking application written in Go. It is deliberately **not** built on a web framework or an ORM. It uses only the Go standard library (`database/sql`, `bufio`, `sync`, `time`) plus the `pgx/v5` PostgreSQL driver. The goal is to teach backend engineering by removing the “magic” that frameworks hide, so that every decision you make is explicit and inspectable.

### 1.2 Why This Project Matters

Most beginner code stops at “the program works.” Real systems stop at “the program works *and* the money is right, the audit trail is complete, and the system stays safe under load and attack.” This project targets the three things that separate a toy from a banking backend:

| Concern | What it means in a bank | How this project addresses it |
|---|---|---|
| **Auditability** | Money cannot disappear into an unrecorded balance change. | An **append-only ledger** (`ledger` table) that records every balance-affecting event. The balance is never edited in place; it is derived from a chain of immutable transactions. |
| **Data integrity** | Two transfers cannot overdraw the same account; corrupted writes must never leave the system half-updated. | `NUMERIC(12,2)` for money (no floating-point rounding), relational foreign keys, and **database transactions** for multi-step operations. |
| **Backend resilience** | Concurrency, retries, and connection management must not corrupt state. | Connection pooling, `SELECT ... FOR UPDATE` row locking, and explicit `Begin`/`Commit`/`Rollback` control. |

### 1.3 Learning Objectives

By reading and extending this project, you will learn:

1. **Relational modeling for money**: why `BIGSERIAL`, foreign keys, and constraints matter more than application-level “checks”.
2. **ACID in practice**: wrapping a transfer into a `BEGIN … COMMIT` block and understanding what `ROLLBACK` buys you.
3. **Go + `database/sql`**: connection pools, `Exec`/`Query` parameter binding with `$1, $2`, and how to structure a data-access layer.
4. **Concurrency safety**: `SELECT ... FOR UPDATE` to serialize access to a row, and why two goroutines cannot both mutate the same account at the same time.
5. **Security fundamentals**: parameterized queries as the primary defense against SQL injection, plus input validation in the CLI layer.
6. **Clean architecture at the file level**: separating setup (`cmd/`), domain (`internal/model/`), business rules (`internal/bank/`), and persistence (`internal/database/`).

> **Note on the current state:** The repository is being built toward this architecture. `internal/db.go` currently contains the placeholder for the PostgreSQL data-access layer, which is intentionally left as a documented contract for the rest of the codebase. The layout and design described here are the target, and each section explains how to get from the current state to the final state.

---

## 2. Architectural Design & Package Layout

### 2.1 Why “No Framework”?

Frameworks provide routing, serialization, and ORMs that “just work.” That convenience hides the plumbing: how a connection is made, how a query is parameterized, how a transaction is committed, and how concurrency is handled. By writing these pieces manually, every part of the stack is visible and inspectable. This project follows the same principle at the module level: the filesystem is the architecture document.

### 2.2 Package Responsibilities

```
mybank/
├── cmd/
│   └── main.go                  # Presentation layer: CLI input/output and driver
├── internal/
│   ├── model/
│   │   └── model.go             # Domain structs: Account, AccountHolder, Transctions
│   ├── bank/
│   │   └── bank.go              # Business rules: create account, deposit, withdraw, transfer
│   └── database/
│       └── db.go                # Persistence: PostgreSQL connection, queries, transactions
└── README.md                    # Quick-start overview
```

### 2.3 Separation of Concerns

| Layer | Package | Owns | Never touches |
|---|---|---|---|
| **Presentation** | `cmd/` | Reading user input, printing results, driving the REPL loop | Any database or money math |
| **Domain** | `internal/model/` | Business entities and their shapes | Database tables, SQL, connection pools |
| **Business logic** | `internal/bank/` | Rules like “an account must have a unique ID”, “transfer moves money”, “withdrawals can’t exceed balance” | SQL and connection details |
| **Data access** | `internal/database/` | Opening connections, pooling, parameterized queries, `BEGIN/COMMIT/ROLLBACK`, row locking | Business rules beyond persistence |

This keeps each layer independently testable. You can test business logic with mock accounts, and test persistence with an integration test against a scratch database, without crossing layer boundaries.

### 2.4 ASCII Architecture Diagram

```
                     +-----------------------+
                     |   cmd/main.go         |
                     |  (CLI: input/output)  |
                     +-----------+-----------+
                                 |
                                 v
                     +-----------------------+
                     |  internal/bank/bank.go|
                     | (business rules)      |
                     +-----------+-----------+
                                 |
                                 v
                     +-----------------------+
                     | internal/model/       |
                     | (domain structs)      |
                     +-----------+-----------+
                                 |
                                 v
                     +-----------------------+
                     | internal/database/    |
                     | db.go (PostgreSQL)    |
                     +-----------------------+
```

### 2.5 Data Flow for a Fund Transfer

1. `cmd/main.go` reads “transfer” and the two account IDs + amount from the user.
2. `bank.*Transfer(...)` enforces business rules (non-negative amount, valid accounts, enough balance).
3. `database.*TransferTx(...)` opens a transaction, locks both rows, performs two ledger inserts (debit + credit), and commits.
4. On any failure, `ROLLBACK` restores the system to the exact state it was in before the transaction started.

---

## 3. PostgreSQL Schema & Design Rationale

### 3.1 Schema Overview

```sql
-- The accounts table holds the financial identity of every customer.
CREATE TABLE accounts (
    id            BIGSERIAL PRIMARY KEY,
    holder_name   TEXT           NOT NULL,
    identifier    TEXT           NOT NULL UNIQUE,
    date_of_birth DATE,
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- The ledger table is append-only by design.
-- Balance is derived by summing this table; rows are never modified.
CREATE TABLE ledger (
    id          BIGSERIAL PRIMARY KEY,
    account_id  BIGINT         NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    amount      NUMERIC(12, 2) NOT NULL,
    type        TEXT           NOT NULL CHECK (type IN ('deposit', 'withdrawal', 'transfer_out', 'transfer_in')),
    description TEXT,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ledger_account ON ledger (account_id);
```

### 3.2 Column-by-Column Rationale

#### `accounts`

| Column | Type | Reason |
|---|---|---|
| `id` | `BIGSERIAL` | A 64-bit auto-incrementing primary key. It fits far more accounts than `INT`, and `SERIAL` guarantees uniqueness without application-side counters. `BIGSERIAL` also survives merge/split events and sharding better than a manual counter. |
| `holder_name` | `TEXT NOT NULL` | The account owner’s name. Kept as free text here because the domain cares about the human, not a lookup key. |
| `identifier` | `TEXT NOT NULL UNIQUE` | A natural key (e.g., national ID, account number, or email). `UNIQUE` prevents duplicate records and lets `WHERE identifier = $1` resolve one row. |
| `date_of_birth` | `DATE` | Stored as a date, not a string, so the database can validate and index it. |
| `created_at` / `updated_at` | `TIMESTAMPTZ` | Time-zone-aware timestamps. `NOW()` is applied on the server, so every row carries an authoritative insert time. `updated_at` supports change tracking without changing the schema. |

#### `ledger`

| Column | Type | Reason |
|---|---|---|
| `id` | `BIGSERIAL` | The ledger is an independent event store. Each row is an immutable event in a sequence. |
| `account_id` | `BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE` | A **foreign key** guarantees that every ledger row belongs to an existing account, and cascade deletion keeps the two tables consistent. Money cannot be recorded against a deleted or nonexistent account. |
| `amount` | `NUMERIC(12, 2) NOT NULL` | **This is the financial-critical column.** `NUMERIC` stores exact decimal values instead of binary floats. `12, 2` reserves 12 digits total, 2 after the decimal point — enough for balances and transfers in a single currency (e.g., up to 999,999,999,999.99). Floating-point arithmetic (`float64`) introduces rounding errors like `0.1 + 0.2 != 0.3`; `NUMERIC` does not. |
| `type` | `TEXT NOT NULL CHECK (...)` | Enumerates the transaction kind. The `CHECK` constraint enforces it at the database level, so even direct SQL cannot insert a bad type. |
| `description` | `TEXT` | Optional detail (e.g., “Transfer to account 42”). Keeps the audit trail self-explanatory without a separate notes table. |
| `created_at` | `TIMESTAMPTZ` | Append-only timestamp. Once written, an event is never modified — this is the foundation of the audit trail. |

### 3.3 Why the Ledger Must Be Immutable

In accounting, a **double-entry ledger** records every change as a distinct event. The balance of an account is not a field you edit; it is the *sum* of all prior events. This design is mandatory for banking for four reasons:

1. **Non-repudiation**: If a balance is stored as a single column, there is no way to prove what changed and when. An append-only ledger records every change with a permanent timestamp and amount.
2. **Auditable corrections**: Correcting an error means adding a new transaction (`-50.00 “Fix error 1234”`) rather than rewriting history. Rewriting history breaks downstream reconciliation and tax reporting.
3. **Referential integrity**: The ledger’s foreign key ensures every debit/credit is attached to a valid account, so no orphaned or fraudulent entries can be inserted.
4. **Reconciliation**: Batch jobs and auditors can replay the ledger, summing events per account, and compare the result against the reported balance. Immutable logs make this exact and reproducible.

---

## 4. Core Engineering Concepts Mastered

### 4.1 Relational Connectivity & Connection Pooling

Go’s `database/sql` package manages a **connection pool** automatically. In production you configure its size and behavior so a burst of concurrent requests (or CLI commands) does not exhaust the database:

```go
db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatal(err)
}

db.SetMaxOpenConns(25)   // max active connections
db.SetMaxIdleConns(5)    // connections kept open for reuse
db.SetConnMaxLifetime(5 * time.Minute) // recycle old connections
```

| Setting | What it does | Typical use |
|---|---|---|
| `SetMaxOpenConns(n)` | Caps how many connections can be in use at once | Prevents the DB from being overwhelmed |
| `SetMaxIdleConns(n)` | Keeps spare connections warm so `Open` is cheap on a hot path | Reduces latency for the next request |
| `SetConnMaxLifetime(d)` | Forces a connection to be closed and replaced after `d` | Protects against stale sockets after a restart or a proxy drop |

> **Note on the current state:** The existing `internal/db.go` is the placeholder for the persistence layer. When it is wired up, `sql.Open` is called once at startup, and every command path uses that pool — no process-level database handle is created per request.

### 4.2 ACID Principles & Transactional Safety

`BEGIN` opens a transaction; `COMMIT` makes its changes permanent; `ROLLBACK` throws them away. In Go this looks like:

```go
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback() // always clean up; ignored on Commit

// ... debit one account, credit another, insert ledger rows ...

if err := tx.Commit(); err != nil {
    return err // Rollback will be applied here if Commit failed
}
```

**Why a transfer needs a transaction:** A transfer is two writes (debit the sender, credit the recipient). If the process crashes between them, the system is left with a debited account that never received credit. `BEGIN … COMMIT` guarantees the two writes happen atomically. `defer tx.Rollback()` is a safety net: even if the code after `Begin` panics or hits an unexpected error, the transaction is cleaned up instead of left open.

**What ACID guarantees here:**

- **Atomicity**: both writes succeed or neither does.
- **Consistency**: the foreign-key and `CHECK` constraints mean the database is always in a valid state.
- **Isolation**: the `SELECT ... FOR UPDATE` in the next section keeps concurrent transfers from stepping on each other.
- **Durability**: once `COMMIT` returns, the changes are on disk (subject to the database’s own durability guarantees).

### 4.3 Concurrency Control & Race Condition Prevention

Two users transfer money **out of the same account** at the same time. Without coordination, both read the balance of `100.00`, both subtract `60.00`, and both write `40.00` — the account is now negative, and the second transfer should have failed.

PostgreSQL solves this with **row-level locking**:

```sql
SELECT balance FROM accounts WHERE id = $1 FOR UPDATE;
```

`FOR UPDATE` blocks the row until the transaction finishes, so the second transaction must wait (or fail with a serialization error). Combined with a transaction it becomes:

```go
var currentBalance string
err := tx.QueryRow(
    `SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`,
    accountID,
).Scan(&currentBalance)
```

This is important in Go because it means correctness does **not** depend on application-level locking (`sync.Mutex`) or on hoping the database is quiet between checks. The database serializes the actual writes, and the Go layer can retry or reject when forced to wait.

### 4.4 Secure Parameterization & SQL Injection Defense

The single most important rule in this codebase: **never build SQL by concatenation.** A string built from user input like:

```go
// DANGEROUS — injectable
query := "SELECT * FROM accounts WHERE id = " + userInput
```

lets a user inject `1 OR 1=1` (or worse) into the query text. Go’s `database/sql` prevents this entirely with **parameterized queries**, where values are sent separately from the SQL:

```go
// SAFE — parameterized
err := db.QueryRow(
    `SELECT id, balance FROM accounts WHERE id = $1`,
    userID,
).Scan(&id, &balance)
```

| Construct | Safe? | Why |
|---|---|---|
| `$1, $2` placeholders | ✅ | Values are bound at the driver level, never parsed as SQL |
| `fmt.Sprintf("... %s ...", input)` | ❌ | Input becomes part of the SQL text |
| `QueryRow("... WHERE id = $1", id)` | ✅ | Value is passed as a parameter |
| `Exec("... INSERT ... $1, $2", a, b)` | ✅ | Same mechanism, for writes |

Because `db.QueryRow` and `tx.Exec` accept value arguments directly, the application cannot accidentally send user input as SQL. Parameterization is the **primary** defense against SQL injection; input validation in the CLI layer is a secondary, UX-focused defense.

### 4.5 Money Handling in Go

- Use `string` or `math/big` for input; never `float64` for monetary amounts.
- Store values as `NUMERIC(12, 2)` in PostgreSQL so the database agrees with the application.
- Format output with `fmt.Sprintf("%.2f", ...)` or a dedicated currency formatter before printing.
- Always round to two decimal places before committing a ledger event, and reject amounts with more than two decimal places with a clear error.

---

## 5. Developer Setup & Usage Guide

### 5.1 Prerequisites

- **Go 1.26+** (the module file declares `go 1.26.0`).
- **PostgreSQL 14+** with the `pgx/v5` driver.
- A terminal that supports ANSI colors (optional but recommended).

### 5.2 Clone & Initialize

```bash
git clone <repository-url>
cd mybank

# Go 1.26+ toolchain automatically initializes and downloads modules.
# (go.mod already pins the Go version; go.sum records checksums.)
go mod tidy
```

> `go.sum` is committed in this repository. It guarantees that the module cache used by anyone rebuilding the project matches the original build. You never need to commit `.cache` or `go.mod` changes by hand.

### 5.3 Configure the Database

Create a dedicated database and record the connection string as an environment variable. Never commit a `.env` file with real credentials.

```bash
# 1. Create the database
createdb mybank

# 2. (Optional) A local .env file for dev only
# DATABASE_URL=postgres://mybank_user:mybank_password@localhost:5432/mybank?sslmode=disable
export DATABASE_URL="postgres://mybank_user:mybank_password@localhost:5432/mybank?sslmode=disable"
```

**Connection string fields:**

| Parameter | Meaning |
|---|---|
| `postgres://user:pass@host:5432/dbname` | Host, port, user, password, database |
| `sslmode=disable` | Local/CI development; use `verify-full` for production |
| `?connect_timeout=5` | Fail fast if the DB is unreachable |

If you use a `.env` file, load it with a tool such as `direnv` or `source .env` before running the CLI.

### 5.4 Run the CLI

```bash
go run cmd/main.go
```

#### Example session

```
Enter 1 to Create Account and continue
Enter 2 to exit
enter 3 to deposit
1
Enter full name: Ada Lovelace
Enter ID number: 12345
Enter your Date of Birth: 1815-12-10
Account created succesfully for user with ID: 1 for Ada Lovelace
```

The current CLI supports **account creation** as the documented starting point. Future commands to be layered on top (deposit, withdraw, transfer) follow the same pattern: read input, call the `bank.*` rule, persist via the `database.*` data-access layer, and print the result.

### 5.5 Build a Release Binary

```bash
go build -o mybank cmd/main.go
./mybank
```

### 5.6 Add the Schema & Wire Up Persistence

The `internal/database/` layer is documented as the contract for the PostgreSQL access layer. The steps to complete it are:

1. **Connect**: open the pool once in `cmd/main.go` and pass it to `internal/bank` as needed.
2. **Create tables**: run the `schema.sql` from Section 3 during startup or setup.
3. **Implement transactions**: use `db.Begin()` / `tx.Commit()` / `tx.Rollback()` for transfers.
4. **Lock rows**: use `SELECT ... FOR UPDATE` inside the transaction before any balance read or write.
5. **Parameterize every query**: `$1, $2, ...` placeholders for all values.
6. **Test against a scratch DB**: run the CLI against `createdb mybank_test` only; never against production.

### 5.7 Testing Checklist

- [ ] `go build ./...` passes.
- [ ] A scratch PostgreSQL database works with the above `DATABASE_URL`.
- [ ] Account creation succeeds and reads back the correct fields.
- [ ] Depositing and withdrawing adjusts the ledger, not a mutable balance column.
- [ ] Two concurrent transfers out of one account are serialized by `FOR UPDATE`.
- [ ] A transfer that would overdraw is rejected with `ROLLBACK`.
- [ ] Every query uses `?`-style/`$1` placeholders — no string concatenation.

---

## Appendix A — Key Design Choices (Summary)

| Choice | Decision | Why |
|---|---|---|
| `NUMERIC(12, 2)` vs `float64` | `NUMERIC(12, 2)` | Exact decimal arithmetic; no rounding drift across deposits/withdrawals/transfers. |
| Balance stored anywhere? | **No** | Balances are derived from the append-only `ledger`. |
| Ledger updates | Append-only inserts | Permanent audit trail; corrections add new events. |
| Idempotency | `nextID` in Go as a *starting point* | Application counter is a placeholder; PostgreSQL `BIGSERIAL` is the source of truth. |
| Transactions | `Begin`/`Commit`/`Rollback` per operation | Atomicity for multi-write operations like transfers. |
| Concurrency | `SELECT ... FOR UPDATE` | Serializes writes to the same account row at the database level. |
| Security | `$1, $2` parameters everywhere | Zero SQL injection surface. |
| Frameworks | None | Standard library + `pgx/v5` only; every layer is explicit. |

---

## Appendix B — Starter `go.mod` Reference

```go
module mybank

go 1.26.0

require (
	github.com/jackc/pgx/v5 v5.5.5
)
```

> Add any extra dependencies (such as `github.com/joho/godotenv` for `.env` loading) with `go get`, then commit `go.sum`.
