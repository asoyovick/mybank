# MyBank

A command-line banking system written in Go, built to master core backend engineering, relational database concepts (PostgreSQL), ACID compliance, and secure API/backend architecture — without heavy framework abstractions.

This project is intentionally scaffolded around four layers:

```
mybank/
├── cmd/
│   └── main.go              # Presentation layer: CLI input & output
├── internal/
│   ├── model/               # Domain structs
│   ├── bank/                # Business logic
│   └── database/            # PostgreSQL persistence
└── docs/
    └── technical-documentation.md   # Full architecture & engineering guide
```

---

## What You Can Do

- **Create an account** from the CLI.
- **Deposit** and **withdraw** money against an account.
- **Transfer** funds to another user on the platform.
- **Audit every change**: all balance-affecting operations are recorded in an append-only `ledger` table, so balances are derived rather than mutated.

> **Note on current scope:** `cmd/main.go` currently implements account creation, and `internal/db.go` is the placeholder for the PostgreSQL data-access layer. The setup guide below shows how to wire the schema and persistence on top.

---

## Project Layout

| Path | Responsibility |
|---|---|
| `cmd/main.go` | Presentation layer: reads CLI input, prints results, drives the session. |
| `internal/model/` | Domain structs: `AccountHolder`, `Account`, `Transctions`. |
| `internal/bank/` | Business rules: account creation, deposit, withdrawal, transfer. |
| `internal/database/db.go` | PostgreSQL connectivity: connection pool, parameterized queries, transactions, row locking.  *(placeholder — wiring guide in `docs/technical-documentation.md`)* |
| `docs/technical-documentation.md` | Comprehensive guide: architecture, schema, ACID, concurrency, security, developer setup. |

---

## Tech Stack

- **Go 1.26+** (standard library only)
- **`database/sql`** + **`github.com/jackc/pgx/v5/stdlib`** (PostgreSQL driver)
- **PostgreSQL 14+** as the relational store

> The project deliberately avoids web frameworks, ORMs, and DI containers so every layer — connection pooling, query parameterization, transactions, and row locking — is explicit and inspectable.

---

## Quick Start

### 1. Clone & initialize

```bash
git clone <repository-url>
cd mybank
go mod tidy
```

Go will resolve `go.mod`, update `go.sum`, and download dependencies.

### 2. Configure the database

Create a database and set the connection string as an environment variable. **Never commit credentials.**

```bash
createdb mybank

# For local development only — load before running the CLI.
export DATABASE_URL="postgres://mybank_user:mybank_password@localhost:5432/mybank?sslmode=disable"
```

| Parameter | Meaning |
|---|---|
| `postgres://user:pass@host:5432/dbname` | Connection details |
| `sslmode=disable` | Local/CI development; use `verify-full` in production |
| `connect_timeout=5` | Fail fast if the database is unreachable |

### 3. Run the CLI

```bash
go run cmd/main.go
```

Example session:

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

### 4. Build a release binary

```bash
go build -o mybank cmd/main.go
./mybank
```

---

## Setup Checklist

- [ ] `go build ./...` passes.
- [ ] PostgreSQL 14+ is running.
- [ ] `createdb mybank` succeeds.
- [ ] `DATABASE_URL` is set (or loaded from `.env`).
- [ ] Database migrations are applied (`accounts`, `ledger` tables).
- [ ] End-to-end CLI session works: create → deposit → withdraw → transfer.

---

## Documentation

- [`docs/technical-documentation.md`](docs/technical-documentation.md) — full architecture, schema rationale, ACID & concurrency, SQL injection defense, and step-by-step developer setup.

---

## Learning Objectives

- Relational modeling for money (`BIGSERIAL`, foreign keys, `NUMERIC(12,2)`).
- ACID in practice (`BEGIN` / `COMMIT` / `ROLLBACK`).
- Go connection pooling and parameterized queries (`$1, $2`).
- Concurrency safety via `SELECT ... FOR UPDATE`.
- SQL injection defense through parameterized statements.
- Clean architecture at the file level.
