# MyBank — End-to-End Architecture & Execution Flow

> **Scope:** This document is the definitive architectural reference for the Go banking system. It describes how a user request moves from the CLI presentation layer, through domain and business rules, into PostgreSQL with transactional integrity, concurrency control, and an immutable audit ledger.

---

## 1. System Overview & Architectural Principles

### 1.1 Architectural Style

MyBank follows **clean architecture** at the file level: each layer has a single, well-defined responsibility, and dependencies point inward.

```
                      +-----------------------+
                      |   cmd/main.go         |
                      | (Presentation/Port)   |
                      +-----------+-----------+
                                  |
                                  v
                      +-----------------------+
                      | internal/bank/bank.go |
                      | (Business Rules)      |
                      +-----------+-----------+
                                  |
                                  v
                      +-----------------------+
                      | internal/model/       |
                      | (Domain Entities)     |
                      +-----------+-----------+
                                  |
                                  v
                      +-----------------------+
                      | internal/database/    |
                      | db.go (PostgreSQL)    |
                      +-----------------------+
```

- **Presentation layer (`cmd/`)** owns only input parsing and output formatting. It knows nothing about SQL, connection pools, or database rows.
- **Business logic (`internal/bank/`)** owns invariants, state transitions, and orchestration. It never writes SQL.
- **Domain (`internal/model/`)** owns structs and value types. It is the vocabulary of the system.
- **Data access (`internal/database/`)** owns the PostgreSQL connection pool, parameterized queries, transactions, and row locking.

This decoupling means the CLI can be swapped for a REST API or a worker process without touching business rules or SQL. Production readiness depends on this boundary holding.

### 1.2 Separation of Concerns

| Layer | Owns | Never touches |
|---|---|---|
| `cmd/` | Reading user input via `bufio`, printing results, driving the REPL loop | Money math, ledger, SQL |
| `internal/bank/` | Account creation, deposit, withdrawal, transfer order, validation | SQL, connection details |
| `internal/model/` | `Account`, `AccountHolder`, `Transctions` structs, value semantics | Database tables, drivers |
| `internal/database/` | Connection pooling, `Exec`/`QueryRow`, `BEGIN`/`COMMIT`/`ROLLBACK`, `SELECT ... FOR UPDATE`, ledger inserts | Business rules, formatting output |

### 1.3 Concurrency & Thread Safety

Go's `database/sql` connection pool is the primary concurrency primitive. The application does not maintain its own per-request database handle, so every goroutine shares one safe pool.

- **Connection pool:** `SetMaxOpenConns`, `SetMaxIdleConns`, and `SetConnMaxLifetime` bound how many sockets are alive at once, preventing exhaustion under load.
- **Row-level locking:** `SELECT ... FOR UPDATE` serializes writes to a single account row inside a transaction, so two goroutines cannot both debit the same account at the same time.
- **No shared mutable state:** The current CLI holds account state in `Bank.accounts` (an in-memory map). That map is a *presentational* placeholder; the production design derives state from the database. Where shared state exists, it is guarded with the database as the single source of truth.
- **Result:** transfers, deposits, and withdrawals are safe when called concurrently; correctness is enforced by the database, not by application-level locks.

---

## 2. End-to-End Request Lifecycle & Execution Flows

### 2.1 Account Creation Flow

```
User types 1
   |
   v
Read full name, ID number, date of birth (bufio)
   |
   v
Bank.CreateAccount(fullname, idno, dob)
   |
   +--> Validate dob as a real date; compute age
   |
   +--> Generate unique account ID
   |
   +--> Insert row into accounts (parameterized: $1, $2, $3)
   |
   +--> Insert ledger row: type = 'deposit', amount = 0.00 (audit entry)
   |
   v
Return account ID to CLI
```

**Pseudocode for the persistence step (the production contract for `internal/database/`):**

```go
func (d *DB) CreateAccount(h Holder) (int64, error) {
    tx, err := d.Begin()
    if err != nil {
        return 0, err
    }
    defer tx.Rollback() // always released; no-op on success

    // 1. Insert the account row
    res, err := tx.Exec(`
        INSERT INTO accounts (holder_name, identifier, date_of_birth, created_at, updated_at)
        VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id
    `, h.Name, h.Identifier, h.DOB)
    if err != nil {
        return 0, err
    }
    id, err := res.LastInsertId()
    if err != nil {
        return 0, err
    }

    // 2. Append the immutable audit ledger entry
    if _, err := tx.Exec(`
        INSERT INTO ledger (account_id, amount, type, description, created_at)
        VALUES ($1, 0.00, 'deposit', 'Initial account created', NOW())
    `, id); err != nil {
        return 0, err
    }

    if err := tx.Commit(); err != nil {
        return 0, err
    }
    return id, nil
}
```

The creation flow has three consistency properties:

1. **Parameterization:** every value is bound as `$1, $2, ...`, so injection is impossible.
2. **Transaction:** the account insert and ledger insert happen in one `BEGIN`/`COMMIT`, so either both succeed or the whole operation is rolled back. There is never an account with no audit entry.
3. **Auditability:** the very first ledger row is recorded, establishing the immutable event chain that the rest of the system appends to.

### 2.2 Deposit & Withdrawal Flows

Both follow the same shape. Deposits append a credit; withdrawals append a debit and enforce a balance invariant.

```
Deposit/Withdrawal request
   |
   v
1. Read account state (SELECT ... FOR UPDATE)
   |
   v
2. Apply validation
   |
   +--> Deposit: amount > 0, within NUMERIC(12,2) precision
   +--> Withdrawal: amount > 0 AND balance >= amount
   |
   v
3. Append ledger entry (immutable; no direct balance UPDATE)
   |
   v
4. COMMIT
   |
   v
Return new balance derived from ledger
```

**Withdrawal pseudocode (invariants + append-only logging):**

```go
func (d *DB) Withdraw(accountID int64, amount string) (string, error) {
    tx, err := d.Begin()
    if err != nil {
        return "", err
    }
    defer tx.Rollback()

    // 1. Lock the row and read balance
    var balance string
    if err := tx.QueryRow(`
        SELECT balance FROM accounts WHERE id = $1 FOR UPDATE
    `, accountID).Scan(&balance); err != nil {
        return "", err
    }

    // 2. Enforce the domain invariant
    current := parseMoney(balance)
    requested := parseMoney(amount)
    if requested <= 0 {
        return "", ErrNonPositiveAmount
    }
    if requested > current {
        return "", ErrInsufficientFunds
    }

    // 3. Append the immutable ledger entry (do NOT UPDATE balance here)
    if _, err := tx.Exec(`
        INSERT INTO ledger (account_id, amount, type, description, created_at)
        VALUES ($1, $2, 'withdrawal', 'Withdrawal of $2', NOW())
    `, accountID, amount); err != nil {
        return "", err
    }

    // 4. Commit the ledger append
    if err := tx.Commit(); err != nil {
        return "", err
    }

    // 5. Return the balance derived by summing the ledger
    return d.deriveBalance(accountID)
}
```

**Why balances are never `UPDATE`-d directly:** storing a balance as a single mutable column is space-efficient but destroys auditability. If a balance changes without a log entry, there is no way to prove what happened or to reconcile. The production design treats the ledger as the canonical source: every debit and credit is a permanent event, and the current balance is the deterministic sum of those events.

### 2.3 Fund Transfer Flow (ACID & Concurrency)

A transfer is the most safety-critical path. Two rows must be locked, two accounts mutated (via ledger appends), and the whole operation atomic.

```
Both accounts enter the same transaction
   |
   v
1. BEGIN
   |
   v
2. SELECT ... FOR UPDATE on sender (locks sender row)
   |
   v
3. SELECT ... FOR UPDATE on recipient (locks recipient row)
   |
   v
4. Validate: amount > 0, sender.balance >= amount
   |
   v
5. Append ledger: 'transfer_out' for sender
   |
   v
6. Append ledger: 'transfer_in' for recipient (same tx)
   |
   v
7. COMMIT
   |
   v
Both updates persist atomically; no partial state remains
```

**PostgreSQL transaction pseudocode:**

```sql
BEGIN;

-- Sender: lock the row before reading balance
SELECT balance FROM accounts WHERE id = $1 FOR UPDATE;
-- (validate amount > 0 and balance >= amount)

-- Append immutable sender ledger entry
INSERT INTO ledger (account_id, amount, type, description, created_at)
VALUES ($1, $2, 'transfer_out', 'Pays $2 to account $3', NOW());

-- Recipient ledger entry
INSERT INTO ledger (account_id, amount, type, description, created_at)
VALUES ($3, $2, 'transfer_in', 'Received $2 from account $1', NOW());

COMMIT;
```

**Why `SELECT ... FOR UPDATE` is mandatory:** without it, two concurrent transfers out of the same sender would both read the same balance, both write a ledger entry, and the sender would be overdrawn. The database-level lock serializes writers to the row so that the second transaction sees the updated balance and either waits or fails with a serialization error.

**Deadlock prevention:**

- Always lock rows in a **consistent order** (e.g., lower account ID first).
- Keep the transaction as short as possible; do not perform HTTP calls or sleeps inside a transaction.
- Use `SELECT ... FOR UPDATE` instead of application-level mutexes, because the database serializes at the point that actually matters.

**Commit/Rollback discipline:**

```go
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback() // clean up if Commit fails or we panic

// ... all writes ...

if err := tx.Commit(); err != nil {
    return err // Deferred Rollback runs if Commit errored
}
```

### 2.4 Transaction and Consistency Guarantees (ACID)

| Property | How it is enforced in MyBank |
|---|---|
| **Atomicity** | `BEGIN` opens a transaction; `COMMIT` finalizes it; `defer tx.Rollback()` + checked `Commit` guarantee all-or-nothing. |
| **Consistency** | Foreign keys, `CHECK` constraints on `ledger.type`, and `NUMERIC` money ensure the row layout is always valid. |
| **Isolation** | `SELECT ... FOR UPDATE` serializes writes to the same row; transactions prevent cross-GoRoutine interference on shared state. |
| **Durability** | `COMMIT` returns only after the WAL is flushed; the append-only ledger survives crashes. |

---

## 3. Data Integrity & Financial Engineering Standards

### 3.1 Why `NUMERIC(12,2)` Instead of `float64`

Financial calculations must be **exact by definition**. IEEE-754 floating-point values such as `0.1` and `0.2` are not exactly representable in binary, so:

```go
0.1 + 0.2 == 0.30000000000000004  // float64
```

A `float64` balance accumulates this drift over thousands of deposits and transfers, and a rounding error in a bank ledger is a correctness failure with legal and customer-trust consequences.

`NUMERIC(12,2)` stores decimal values **exactly**:

```sql
ALTER TABLE accounts ADD COLUMN balance NUMERIC(12, 2) NOT NULL DEFAULT 0.00;
```

- `12` is the total number of digits.
- `2` is the number of digits after the decimal point.
- The database performs arithmetic in exact decimal arithmetic, so `100.00 - 30.50` is `69.50`, not a repeating binary fraction.

**Rule:** money is represented as a `string` in Go, parsed into `NUMERIC` at the database boundary, and formatted back to `string` before printing.

### 3.2 The Append-Only Ledger Pattern

The `ledger` table is an **event log**. Every balance-affecting action is an immutable row, and the current balance is derived by summing events.

```
      +-------------------+
      |  accounts (1 row) |
      +--------+----------+
               | 1
               | many
               v
      +-------------------+
      |    ledger (N rows)|
      |  +-------------+  |
      |  | deposit  10 |  |
      |  | withdrawal 4|  |
      |  | transfer_in 3|  |
      |  +-------------+  |
      +-------------------+
```

**Why balances are never updated in isolation:**

1. **Auditability:** every transaction has a permanent, timestamped record. Correcting an error means appending a compensating entry, never rewriting history.
2. **Reconciliation:** batch jobs and auditors sum the ledger per account and compare against the reported balance.
3. **Non-repudiation:** a row that cannot be modified cannot be disputed.
4. **Consistency under concurrency:** an `UPDATE accounts SET balance = ...` relies on in-place writes; appending a ledger row is an insertion, which PostgreSQL handles with far fewer write-path hazards.

The ledger is the single durable record of money movement; the `accounts` row is a convenience aggregate.

---

## 4. Database Schema & Relational Design

### 4.1 Schema

```sql
-- 1:1 relationship from account to its audit trail.
CREATE TABLE accounts (
    id            BIGSERIAL      PRIMARY KEY,
    holder_name   TEXT           NOT NULL,
    identifier    TEXT           NOT NULL UNIQUE,
    date_of_birth DATE,
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- Append-only event log. Balance is derived, not stored.
CREATE TABLE ledger (
    id          BIGSERIAL      PRIMARY KEY,
    account_id  BIGINT         NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    amount      NUMERIC(12, 2) NOT NULL,
    type        TEXT           NOT NULL CHECK (type IN (
        'deposit', 'withdrawal', 'transfer_out', 'transfer_in'
    )),
    description TEXT,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ledger_account_id ON ledger (account_id);
```

### 4.2 Design Rationale for Relational Choices

| Construct | Why it is used |
|---|---|
| `BIGSERIAL` (primary key) | 64-bit auto-incrementing unique identifier. `BIGINT` supports far more accounts than `INT`, and `SERIAL` guarantees uniqueness without an application-side counter. |
| `FOREIGN KEY account_id REFERENCES accounts(id) ON DELETE CASCADE` | Every ledger row belongs to an existing account. Deleting an account cascades to its audit trail so no orphaned money events can exist. |
| `UNIQUE (identifier)` on `accounts` | A natural key (e.g., ID number or email) is enforced unique at the database level; `WHERE identifier = $1` resolves one row. |
| `NUMERIC(12, 2) NOT NULL` | Exact decimal representation of money; no rounding drift. |
| `CHECK (type IN (...))` | Database-enforced enumeration: only valid ledger types can be inserted, even from ad-hoc SQL. |
| `TIMESTAMPTZ NOT NULL DEFAULT NOW()` | Time-zone-aware timestamp set by PostgreSQL; every row carries authoritative insert time. |
| `ON DELETE CASCADE` | Keeps the two tables consistent when an account is removed; no dangling ledger rows. |

### 4.3 Relational Mapping Summary

```
accounts.id ─────────────► ledger.account_id
   (parent)                  (child, NOT NULL FK)
   cascade ON DELETE         prevents orphaned ledger events
```

- One account can have many ledger rows; each ledger row belongs to exactly one account.
- The schema is therefore a **1:N relationship**, enforced by the foreign key, and the ledger is the child that represents the account's immutable history.

---

## Appendix A — Key Invariants

```text
1. Every balance-affecting operation appends exactly one ledger row.
2. balances are derived by summing the ledger, never stored as a single mutable value.
3. All money is NUMERIC(12,2); no float64 anywhere in the persistence path.
4. Transfers are one transaction: BEGIN → lock rows → debit ledger → credit ledger → COMMIT.
5. Every query uses $1, $2 parameterization; no string-concatenated SQL.
6. Row writes are serialized with SELECT ... FOR UPDATE on the affected rows.
```

---

## Appendix B — Data Flow Summary (Single Source of Truth)

```
CLI input (bufio) ──► bank.* rule (invariants) ──► database.* txn (BEGIN/ROLLBACK)
     │                                              │
     │                                              ├─ SELECT ... FOR UPDATE
     │                                              ├─ parameterized INSERT/UPDATE
     │                                              └─ COMMIT/ROLLBACK
     └────────► printed result (formatted string) ───┘
```

The database is the single source of truth. The presentation layer only shapes what the user sees; the business layer decides what may happen; the transaction layer decides whether it persists.
