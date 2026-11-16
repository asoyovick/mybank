MyBank Production Banking System: Extension Documentation
Version 1.0 · Extends the MyBank CLI (Go + PostgreSQL) into a production-grade core banking platform
1. Purpose and Scope
MyBank today is a learning-oriented command-line banking application written in Go with PostgreSQL. It is deliberately framework-free and is organised into four layers: presentation (cmd/), domain models (internal/model), business logic (internal/bank) and persistence (internal/database). Only account creation is currently implemented in the CLI, and the database layer is a placeholder.
This document describes how to evolve that foundation into a system that can run in a real banking environment. It defines requirements, architecture, data model, APIs, security, compliance, operations and a phased roadmap.
Important disclaimer. Software that holds real customer money is subject to financial regulation. Documentation and good engineering are necessary but not sufficient. Before going live you will need a banking or payment-institution licence (or a licensed sponsor/partner bank), an independent security audit, penetration testing, and legal review in your jurisdiction. Items in this document marked [Regulatory] require input from your regulator and legal counsel.
1.1 Goals
• Preserve the correctness guarantees of the current design (ACID, append-only ledger, parameterised SQL).
• Move from a single-user CLI to a multi-user, network-accessible service.
• Add identity, KYC/AML, auditing, observability and resilience.
• Provide a clear path from prototype to production.
1.2 Non-goals (initial release)
• Card issuing and card-scheme processing.
• Foreign-exchange trading.
• Lending and credit scoring (described in the roadmap as a later phase).
1.3 Current State vs Target State
Area
Current (README)
Target (production)
Interface
CLI
REST/JSON API, admin console, optional web and mobile clients
Users
Single operator
Customers, tellers, compliance officers, administrators
Authentication
None
Password + MFA, OAuth2/OIDC, short-lived tokens
Persistence
Placeholder db.go
PostgreSQL with migrations, replication, backups
Ledger
Append-only ledger table (planned)
Double-entry ledger with reconciliation
Compliance
None
KYC, AML screening, audit trail, data protection
Operations
Local run
Containers, CI/CD, monitoring, DR
2. Requirements
2.1 Functional Requirements
Customer and account management
• FR-1: Register a customer with identity details and complete KYC verification before activation.
• FR-2: Open multiple accounts per customer (savings, current) in a defined currency.
• FR-3: Freeze, unfreeze and close accounts, with reason codes and approvals.
• FR-4: Support joint accounts and account mandates (who may transact).
Transactions
• FR-5: Deposit funds (teller, bank transfer, mobile-money or card gateway).
• FR-6: Withdraw funds, subject to available balance and limits.
• FR-7: Internal transfers between accounts, atomically.
• FR-8: External transfers through payment rails (see section 9).
• FR-9: Reverse a transaction through a compensating entry, never by deletion.
• FR-10: Enforce daily, per-transaction and velocity limits.
Reporting
• FR-11: Account statements by date range, exportable as PDF and CSV.
• FR-12: Regulatory and management reports (large-transaction reports, suspicious-activity reports, trial balance).
• FR-13: Notifications (SMS, email, push) for transactions and security events.
Administration
• FR-14: Role-based access for staff with maker-checker approval for sensitive actions.
• FR-15: Full audit log of every read of sensitive data and every state change.
2.2 Non-Functional Requirements
Category
Requirement
Correctness
No lost or duplicated money; ledger always balances to zero
Availability
Target 99.95% for transaction APIs; planned maintenance windows documented
Durability
RPO of at most 5 minutes (target near zero with synchronous replication); RTO of at most 1 hour
Performance
p95 latency under 300 ms for balance and transfer calls at expected peak load
Scalability
Stateless API tier scales horizontally; database scales vertically first, then read replicas, then partitioning
Security
Encryption in transit and at rest, least privilege, secrets management, audit logging
Auditability
Every financial event traceable to an actor, time, source IP and request ID
Maintainability
Automated tests, migrations under version control, documented runbooks
3. Architecture
3.1 Design Principles
1.  The ledger is the source of truth. Balances are derived from, or reconciled against, immutable ledger entries.
2.  Every money movement is a database transaction. Either all entries commit or none do.
3.  Idempotency everywhere. Network retries must never double-post a transaction.
4.  Least privilege. Each service, database role and human gets only the access it needs.
5.  Keep the Go standard-library philosophy where it helps. Use well-vetted libraries for security-critical concerns (JWT, crypto, rate limiting) rather than writing your own.
3.2 Logical Architecture

```text
            Clients (web, mobile, partner systems, admin UI)
                              |
                        HTTPS / TLS 1.2+
                              |
                +---------------------------+
                |  API Gateway / WAF        |  rate limiting, TLS termination
                +---------------------------+
                              |
        +---------------------+----------------------+
        |                     |                      |
  +-----------+        +-------------+        +--------------+
  | Auth      |        | Core        |        | Compliance   |
  | Service   |        | Banking API |        | (KYC / AML)  |
  +-----------+        +-------------+        +--------------+
        |                     |                      |
        |              +-------------+               |
        |              | Ledger      |               |
        |              | module      |               |
        |              +-------------+               |
        |                     |                      |
        +----------+----------+-----------+----------+
                   |                      |
           +---------------+      +----------------+
           | PostgreSQL    |      | Message queue  |
           | primary +     |      | (events,       |
           | replicas      |      | notifications) |
           +---------------+      +----------------+
```

Start as a modular monolith: one deployable Go service with strict internal module boundaries. Split into separate services only when scale or team structure demands it. This preserves transactional integrity, which is far harder across service boundaries.
3.3 Proposed Repository Layout

```text
mybank/
├── cmd/
│   ├── api/                 # HTTP server entry point
│   ├── cli/                 # Existing CLI kept for admin/dev tooling
│   └── worker/              # Background jobs (reconciliation, notifications)
├── internal/
│   ├── model/               # Domain types
│   ├── bank/                # Business rules (use cases)
│   ├── ledger/              # Double-entry posting engine
│   ├── auth/                # Authentication, sessions, RBAC
│   ├── compliance/          # KYC, AML rules, limits
│   ├── database/            # Connection pool, repositories, tx helpers
│   ├── httpapi/             # Handlers, middleware, DTOs
│   ├── notify/              # SMS, email, push
│   └── platform/            # Config, logging, metrics, clock, ID generation
├── migrations/              # Versioned SQL migrations
├── api/                     # OpenAPI specification
├── deploy/                  # Dockerfiles, Kubernetes manifests, Terraform
└── docs/                    # This documentation, runbooks, ADRs
```
3.4 Layer Responsibilities
• httpapi: parses and validates requests, authenticates, maps errors to status codes. Contains no business logic.
• bank: use cases such as Transfer, OpenAccount, Withdraw. Enforces business rules and limits, and decides the transaction boundary.
• ledger: the only package allowed to write ledger entries. Guarantees double-entry balance.
• database: repositories and transaction helpers. All SQL is parameterised.
• model: plain structs with no dependencies. (Rename the current Transctions type to Transaction.)
4. Data Model
4.1 Money Representation
Never use floating point for money. Store amounts as integer minor units (for example cents) in BIGINT, with a separate currency code. This avoids rounding errors and makes arithmetic exact. The current README uses NUMERIC(12,2), which is acceptable for a prototype, but integer minor units are recommended for production because they remove ambiguity across currencies with different decimal places.
4.2 Core Tables

```sql
-- Customers and identity
CREATE TABLE customers (
  id             BIGSERIAL PRIMARY KEY,
  external_ref   UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
  full_name      TEXT NOT NULL,
  date_of_birth  DATE NOT NULL,
  national_id    TEXT NOT NULL,          -- encrypted at application level
  kyc_status     TEXT NOT NULL DEFAULT 'pending'
                 CHECK (kyc_status IN ('pending','verified','rejected','review')),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Accounts
CREATE TABLE accounts (
  id            BIGSERIAL PRIMARY KEY,
  customer_id   BIGINT NOT NULL REFERENCES customers(id),
  account_no    TEXT NOT NULL UNIQUE,
  currency      CHAR(3) NOT NULL,
  type          TEXT NOT NULL CHECK (type IN ('savings','current','internal')),
  status        TEXT NOT NULL DEFAULT 'active'
                CHECK (status IN ('active','frozen','closed')),
  balance_minor BIGINT NOT NULL DEFAULT 0,   -- cached, reconciled against ledger
  version       BIGINT NOT NULL DEFAULT 0,   -- optimistic concurrency
  opened_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (type = 'internal' OR balance_minor >= 0)
);

-- A business event grouping several ledger entries
CREATE TABLE transactions (
  id              BIGSERIAL PRIMARY KEY,
  idempotency_key TEXT NOT NULL UNIQUE,
  kind            TEXT NOT NULL,   -- deposit, withdrawal, transfer, reversal, fee
  status          TEXT NOT NULL CHECK (status IN ('pending','posted','failed','reversed')),
  reference       TEXT,
  reverses_id     BIGINT REFERENCES transactions(id),
  initiated_by    BIGINT,          -- user id
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Immutable double-entry ledger
CREATE TABLE ledger_entries (
  id             BIGSERIAL PRIMARY KEY,
  transaction_id BIGINT NOT NULL REFERENCES transactions(id),
  account_id     BIGINT NOT NULL REFERENCES accounts(id),
  direction      TEXT NOT NULL CHECK (direction IN ('debit','credit')),
  amount_minor   BIGINT NOT NULL CHECK (amount_minor > 0),
  currency       CHAR(3) NOT NULL,
  balance_after  BIGINT NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Append-only audit trail
CREATE TABLE audit_log (
  id          BIGSERIAL PRIMARY KEY,
  actor_id    BIGINT,
  actor_role  TEXT,
  action      TEXT NOT NULL,
  entity      TEXT NOT NULL,
  entity_id   TEXT,
  request_id  TEXT,
  ip_address  INET,
  details     JSONB,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```
4.3 Enforcing Immutability
Revoke UPDATE and DELETE on ledger_entries and audit_log from the application role, and add a trigger that raises an exception on any attempt. Corrections are made only by posting compensating entries linked through reverses_id.
4.4 Double-Entry Rule
Every transaction must satisfy: the sum of debits equals the sum of credits, per currency. A deposit of 100.00 is not a single credit. It is a credit to the customer account and a debit to an internal cash or settlement account. Enforce this in the ledger package and verify nightly with a SQL check:
SELECT transaction_id
FROM ledger_entries
GROUP BY transaction_id, currency
HAVING SUM(CASE WHEN direction='debit' THEN amount_minor ELSE -amount_minor END) <> 0;
-- must return zero rows
4.5 Additional Tables for Later Phases
users and roles (staff and customer logins), devices, limits, beneficiaries, kyc_documents, aml_alerts, notifications, external_payments, reconciliation_runs, fee_schedules.
4.6 Migrations
Use versioned SQL migrations (for example golang-migrate or goose). Rules: migrations are forward-only in production, reviewed like code, tested against a copy of production-shaped data, and never edited after release.
5. Transaction Processing
5.1 Transfer Algorithm
1.  Validate the request and authenticate the caller.
2.  Check the idempotency_key. If the key already exists, return the stored result.
3.  BEGIN a transaction at READ COMMITTED (or SERIALIZABLE for the strictest paths).
4.  Lock both accounts with SELECT ... FOR UPDATE, always in ascending account-id order to prevent deadlocks.
5.  Verify both accounts are active, currencies match, funds are sufficient, and limits are not exceeded.
6.  Insert the transactions row, then the two ledger_entries rows.
7.  Update cached balances and increment version.
8.  Write an audit_log row and an outbox event (see 5.3) in the same transaction.
9.  COMMIT. On any error, ROLLBACK and return a typed error.
func (s *Service) Transfer(ctx context.Context, req TransferRequest) (*Result, error) {
    return s.db.WithTx(ctx, func(tx *sql.Tx) (*Result, error) {
        if res, ok, err := s.repo.FindByIdempotencyKey(ctx, tx, req.Key); err != nil || ok {
            return res, err
        }
        first, second := orderIDs(req.FromID, req.ToID)
        if err := s.repo.LockAccounts(ctx, tx, first, second); err != nil {
            return nil, err
        }
        // validate, post ledger entries, update balances, write audit + outbox
        return s.ledger.PostTransfer(ctx, tx, req)
    })
}
5.2 Idempotency
Clients send an Idempotency-Key header with every money-moving request. The server stores the key with the request hash and the response. A repeat with the same key and same payload returns the original response. A repeat with the same key and a different payload is rejected with 409 Conflict.
5.3 Transactional Outbox
Side effects such as sending an SMS must not happen inside the database transaction. Write an event row to an outbox table in the same transaction, and let a worker publish it afterwards. This guarantees that notifications are sent if and only if the money movement committed.
5.4 Concurrency Control
• Pessimistic row locking (FOR UPDATE) for balance-changing operations.
• Optimistic version checks for low-contention updates such as profile edits.
• Consistent lock ordering and short transactions to avoid deadlocks.
• Retry on serialization failures and deadlocks, bounded and with jitter.
5.5 Holds and Pending States
For card-style or external payments, post a hold (reducing available balance without moving ledger funds), then settle or release it. Track balance_minor (ledger balance) and available_minor (ledger balance minus holds) separately.
5.6 Reversals and Disputes
A reversal creates a new transaction with mirrored entries and reverses_id pointing to the original. The original is marked reversed. History is never rewritten.
5.7 Reconciliation
Run automatically and alert on any difference:
• Internal: for each account, balance_minor equals the sum of its ledger entries.
• Global: total debits equal total credits.
• External: compare daily settlement files from partner banks and payment rails to the ledger, and list unmatched items for investigation.
6. API Design
6.1 Conventions
• REST over HTTPS, JSON bodies, versioned path prefix /api/v1.
• OpenAPI 3 specification kept in api/openapi.yaml and used to generate documentation and client SDKs.
• Money in requests and responses as { "amount_minor": 10050, "currency": "USD" }.
• Every response includes an X-Request-ID. Errors follow RFC 7807 (application/problem+json).
• Cursor-based pagination for lists.
6.2 Endpoints (initial set)
Method
Path
Description
POST
/auth/login
Credentials to short-lived access token plus refresh token
POST
/auth/mfa/verify
Complete second factor
POST
/auth/refresh
Rotate refresh token
POST
/customers
Register customer (starts KYC)
GET
/customers/{id}
Retrieve profile
POST
/accounts
Open account
GET
/accounts/{id}
Balance and status
GET
/accounts/{id}/transactions
Paginated history
GET
/accounts/{id}/statements?from=&to=
Statement export
POST
/transfers
Internal transfer (requires Idempotency-Key)
POST
/deposits
Record or initiate deposit
POST
/withdrawals
Withdraw funds
POST
/transactions/{id}/reverse
Staff-only reversal (maker-checker)
GET
/admin/audit-log
Compliance officers only
6.3 Example: Transfer
POST /api/v1/transfers
Authorization: Bearer <token>
Idempotency-Key: 7f3c1e0a-9a52-4b8e-8b3a-2e5d9c1f0a11
Content-Type: application/json

{
  "from_account": "0012345678",
  "to_account": "0087654321",
  "amount_minor": 25000,
  "currency": "USD",
  "reference": "Rent November"
}
{
  "transaction_id": 90211,
  "status": "posted",
  "from_balance_minor": 175000,
  "created_at": "2026-11-16T09:30:12Z"
}
6.4 Error Handling
Use stable, machine-readable codes such as INSUFFICIENT_FUNDS, ACCOUNT_FROZEN, LIMIT_EXCEEDED, DUPLICATE_REQUEST, KYC_INCOMPLETE. Never leak internal errors, SQL text or stack traces to clients.
7. Security
7.1 Authentication
• Passwords hashed with Argon2id (or bcrypt with a high cost factor), unique salt per user.
• Multi-factor authentication (TOTP or hardware key) mandatory for staff and for customer actions above a risk threshold.
• Short-lived access tokens (5 to 15 minutes) with rotating refresh tokens bound to a device; revoke on logout or suspicious activity.
• Account lockout and progressive delay after repeated failures; credential-stuffing detection.
7.2 Authorisation
• Role-based access control with least privilege: customer, teller, support, compliance, admin.
• Object-level checks on every request, so a customer can only access their own accounts.
• Maker-checker (four-eyes) for reversals, limit changes, account closures and manual adjustments.
• Separate staff and customer identity realms.
7.3 Data Protection
• TLS 1.2 or higher everywhere, including between internal components. Replace the README's sslmode=disable with verify-full in every non-local environment.
• Encryption at rest for databases, backups and object storage.
• Field-level encryption for national IDs and documents, with keys in a KMS or HSM and a rotation policy.
• Never log passwords, tokens, full account numbers or card data. Mask sensitive fields in logs and UIs.
• Secrets from a secrets manager, not environment files committed to Git.
7.4 Application Security
• Parameterised SQL only (already a project principle). Add static analysis (gosec, staticcheck) in CI.
• Strict input validation, request size limits, and timeouts on every handler.
• Rate limiting per IP, per user and per endpoint; stricter on login and transfers.
• Security headers, CORS allow-list, CSRF protection for browser sessions.
• Dependency scanning (govulncheck), pinned versions and a software bill of materials.
7.5 Fraud and Abuse Controls
• Velocity rules (count and amount per window), new-beneficiary cool-downs, unusual-location and device checks.
• Step-up authentication for high-risk actions.
• A rules engine that can hold a transaction for manual review, with alerts to the compliance queue.
7.6 Threat Model Summary
Threat
Mitigation
SQL injection
Parameterised queries, least-privilege DB role
Replay or duplicate submission
Idempotency keys, nonces, token expiry
Race-condition double spend
Row locks, ordered locking, CHECK constraints
Account takeover
MFA, device binding, anomaly detection
Insider abuse
RBAC, maker-checker, immutable audit log, access reviews
Data breach
Encryption, minimisation, network segmentation, monitoring
Ledger tampering
Append-only permissions, triggers, hash chaining of entries (optional)
DDoS
CDN/WAF, rate limiting, autoscaling
8. Compliance and Regulatory Considerations [Regulatory]
Requirements depend heavily on your country and licence type. Identify your regulator early and map each obligation below to a control.
Area
Typical requirement
System support
KYC / CDD
Verify identity before activating accounts; enhanced due diligence for high-risk customers
kyc_status, document store, review workflow, third-party ID verification
AML / CFT
Transaction monitoring, sanctions and PEP screening, suspicious-activity reporting
Rules engine, screening integration, aml_alerts, SAR export
Data protection
Lawful basis, minimisation, subject access, retention limits, breach notification
Data inventory, retention jobs, access logs, export/delete tooling
Payment security
PCI DSS if card data is handled (avoid by using a tokenising provider)
Tokenisation, no raw PAN storage
Auditability
Records retained for a statutory period (commonly 5 to 10 years)
Immutable ledger and audit log, archival storage
Operational resilience
Incident reporting, DR testing, vendor risk management
Runbooks, DR drills, vendor register
Consumer protection
Clear terms, fees disclosure, dispute handling
Fee schedules, dispute workflow, statements
Maintain a compliance matrix in docs/compliance-matrix.md linking each obligation to its control, owner and evidence.
9. Integrations
Integration
Purpose
Notes
Payment rails (national switch, ACH/SEPA/RTGS equivalents)
External transfers
Use a licensed partner or direct membership; handle asynchronous settlement and returns
Mobile-money and card gateways
Deposits and withdrawals
Webhooks must be signed, verified and idempotent
KYC / ID verification provider
Identity checks
Store results and consent; do not over-collect
Sanctions and PEP screening
AML
Screen at onboarding and continuously
SMS, email, push
Notifications
Via outbox; never block the transaction
Credit bureau (optional)
Lending phase
Consent required
Wrap every external system behind an interface in internal/, with timeouts, retries with backoff, circuit breakers, and contract tests against sandbox environments.
10. Infrastructure and Deployment
10.1 Environments
local → dev → staging (production-like, sanitised data) → production. Production data never leaves production, and staging never contains real customer data.
10.2 Runtime
• Containerise with a minimal, non-root image (multi-stage Go build, distroless base).
• Run on Kubernetes or a managed container platform across at least two availability zones.
• Stateless API pods behind a load balancer with health and readiness probes.
• Managed PostgreSQL (or self-managed with Patroni): one primary, synchronous standby, additional read replicas for reporting.
• Connection pooling with PgBouncer; tune database/sql pool settings (SetMaxOpenConns, SetMaxIdleConns, SetConnMaxLifetime).
10.3 CI/CD Pipeline
1.  Lint, go vet, staticcheck, gosec, govulncheck.
2.  Unit tests and integration tests against ephemeral PostgreSQL.
3.  Build and sign the container image; generate SBOM.
4.  Deploy to staging, run smoke and end-to-end tests.
5.  Manual approval, then progressive rollout (canary or blue/green) to production.
6.  Automatic rollback on error-rate or latency regression.
10.4 Configuration
Twelve-factor configuration through environment variables and a secrets manager. Keep DATABASE_URL out of source control, and use separate database credentials for the API (limited DML), migrations (DDL) and reporting (read-only).
11. Observability and Operations
• Logging: structured JSON logs with request ID, user ID (not PII), and transaction ID. Ship to a central, access-controlled store.
• Metrics: request rate, error rate, latency percentiles, DB pool saturation, lock wait time, ledger imbalance count, queue depth. Expose with Prometheus.
• Tracing: OpenTelemetry across API, database and external calls.
• Alerting: page on ledger imbalance, reconciliation break, elevated 5xx, failed backups, unusual login or transaction patterns.
• Runbooks: one page per alert describing diagnosis and remediation.
• Backups and DR: continuous WAL archiving with point-in-time recovery, encrypted off-site copies, quarterly restore tests, and an annual full disaster-recovery exercise.
• Capacity and cost reviews quarterly.
12. Testing Strategy
Level
Scope
Examples
Unit
Business rules, limits, ledger math
Table-driven Go tests for fee and limit logic
Integration
Repositories and transactions against real PostgreSQL
Rollback on failure, constraint enforcement
Concurrency
Races and deadlocks
1,000 parallel transfers between two accounts; total money conserved
Property-based
Ledger invariants
For random operation sequences, debits equal credits and no balance goes negative
API / contract
OpenAPI conformance, auth, idempotency
Duplicate request returns identical response
Security
SAST, DAST, dependency scans, pen test
Annual third-party penetration test
Performance
Load and soak
Peak-day load plus 3x headroom
Chaos and DR
Failure handling
Kill primary DB, drop network to a payment partner
UAT and compliance
Business and regulatory sign-off
Test cases mapped to the compliance matrix
Aim for high coverage on bank, ledger and auth packages, and block merges when the money-conservation tests fail.
13. Roadmap
Phase 0: Foundation (current project completion)
• Implement internal/database (pool, WithTx, repositories) and migrations.
• Complete deposit, withdrawal and transfer in the CLI with the ledger.
• Fix naming (Transctions to Transaction) and add unit and concurrency tests.
• CI pipeline with linting and tests.
Phase 1: Secure API
• HTTP API with OpenAPI spec, authentication, RBAC and MFA.
• Idempotency, audit log, structured logging and metrics.
• Containerisation and staging environment.
Phase 2: Core Banking Hardening
• Double-entry ledger with internal accounts, reconciliation jobs, holds and reversals.
• Limits, fees, statements and notifications through the outbox.
• Maker-checker workflows and admin console.
Phase 3: Compliance and Integrations
• KYC provider and AML screening, suspicious-activity reporting.
• Payment-rail and mobile-money integrations.
• Data-protection tooling, retention and archival.
Phase 4: Production Readiness
• External security audit and penetration test, remediation.
• Performance and DR testing, runbooks, on-call rotation.
• Regulatory approval and pilot with limited customers and caps.
Phase 5: Growth
• Lending, interest accrual, standing orders, multi-currency, partner APIs, analytics.
14. Production Readiness Checklist
[ ] Banking or payment licence, or signed agreement with a licensed partner
[ ] Compliance matrix complete and approved by legal
[ ] All money values stored as integer minor units
[ ] Ledger is double-entry, immutable, and passes nightly balance checks
[ ] Idempotency enforced on every money-moving endpoint
[ ] MFA enforced for staff; maker-checker on sensitive actions
[ ] TLS everywhere, sslmode=verify-full, encryption at rest, KMS-managed keys
[ ] Secrets in a secrets manager; no credentials in the repository
[ ] Independent penetration test passed and findings remediated
[ ] Load test and failover test passed
[ ] Backups verified by restore; RPO/RTO met
[ ] Monitoring, alerting and on-call in place; runbooks reviewed
[ ] Incident response and breach notification procedures approved
[ ] Data retention, subject-access and deletion procedures implemented
[ ] Pilot completed with transaction caps and sign-off
15. Glossary
• ACID: Atomicity, Consistency, Isolation, Durability, the guarantees of a reliable database transaction.
• AML / CFT: Anti-money-laundering and countering the financing of terrorism.
• Double-entry: Recording every movement as equal debits and credits across accounts.
• Idempotency: Repeating a request has the same effect as performing it once.
• KYC / CDD: Know Your Customer and Customer Due Diligence.
• Maker-checker: One person initiates an action and a second person approves it.
• Outbox pattern: Storing events in the same transaction as the data change so they are published reliably afterwards.
• RPO / RTO: Recovery Point Objective and Recovery Time Objective, the acceptable data loss and downtime after a disaster.
• WAF: Web Application Firewall.
16. Team Workflow: Issue-Based Development
Every unit of work, whether a feature, bug, refactor or documentation task, is tracked as an issue. Each contributor works on the issues assigned to them, on their own branch, and delivers through a pull request that closes the issue. The examples use GitHub terminology; GitLab and others map one-to-one.
16.1 Core Rules
1.  No work without an issue. If it is not in the tracker, it does not exist. Create the issue first, then branch.
2.  One issue, one owner, one branch, one pull request. Large features are split into smaller issues linked to a parent (epic) issue.
3.  Small and shippable. An issue should be completable in 1 to 3 days. If it is bigger, split it.
4.  main is protected. No direct pushes. Every change goes through a reviewed pull request with passing checks.
5.  Money-path code needs two reviewers. Anything touching internal/ledger, internal/bank, internal/auth or migrations/ requires two approvals.
6.  The issue is the source of truth for requirements, discussion and decisions. Record decisions in the issue, not in chat.
16.2 Issue Lifecycle
Backlog -> Ready -> In Progress -> In Review -> Done
              |                       |
              +--- Blocked <----------+
Stage
Meaning
Who moves it
Backlog
Captured but not yet refined
Anyone
Ready
Has acceptance criteria, estimate, dependencies cleared
Maintainer
In Progress
Assigned and a branch exists
Assignee
In Review
Pull request open and linked
Assignee
Blocked
Waiting on another issue or decision; blocker is named in a comment
Assignee
Done
Pull request merged, issue auto-closed
Automatic on merge
Use a project board (GitHub Projects or similar) with these columns. Contributors pick work only from Ready, assign themselves, and move the card.
16.3 Labels
Use a small, consistent label set.
Group
Labels
Type
type: feature, type: bug, type: refactor, type: docs, type: test, type: chore, type: security
Area
area: database, area: ledger, area: bank, area: api, area: auth, area: compliance, area: infra, area: cli, area: docs
Priority
P0 critical, P1 high, P2 normal, P3 low
Phase
phase-0, phase-1, phase-2, phase-3, phase-4
State
blocked, needs-design, needs-review, good first issue, regulatory
type: security and regulatory issues are never closed without a maintainer sign-off.
16.4 Feature Issue Template
Save as .github/ISSUE_TEMPLATE/feature.md:
---
name: Feature
about: A new capability for MyBank
labels: "type: feature"
---

## Summary
One or two sentences: what and why.

## Parent / Epic
Part of #<epic-number>

## Scope
- In scope:
- Out of scope:

## Acceptance Criteria
- [ ] Criterion 1 (testable)
- [ ] Criterion 2
- [ ] Unit tests added
- [ ] Integration test against PostgreSQL added (if it touches the database)
- [ ] Documentation updated

## Technical Notes
Packages affected, SQL changes, API endpoints, edge cases.

## Security / Money Impact
Does this move money, change balances, handle PII or alter permissions? If yes, describe the controls.

## Dependencies
Blocked by #<issue>, blocks #<issue>

## Estimate
XS / S / M / L (L must be split)
Create similar templates for bug (steps to reproduce, expected vs actual, logs, severity) and security (private reporting; never post exploit details publicly).
16.5 Branch, Commit and Pull Request Conventions
Branch names: <type>/<issue-number>-<short-description>
feature/12-transfer-with-row-locking
bugfix/27-negative-balance-on-withdraw
chore/5-add-ci-pipeline
Commits follow Conventional Commits and reference the issue:
feat(bank): add transfer use case with ordered locking (#12)
fix(ledger): reject zero-amount entries (#27)
test(bank): add 1000-goroutine transfer race test (#12)
Pull request template (.github/pull_request_template.md):
## What
Closes #<issue-number>

## How
Brief description of the approach.

## Checklist
- [ ] Acceptance criteria met
- [ ] Tests pass locally (`go test ./...`)
- [ ] `go vet` and linter clean
- [ ] Migrations are forward-only and tested
- [ ] No secrets or credentials committed
- [ ] Money-path change: concurrency and rollback tests included
- [ ] Docs / OpenAPI updated
Writing Closes #12 in the pull request description closes the issue automatically on merge.
16.6 Day-to-Day Flow for a Contributor
1.  Pick a Ready issue, assign yourself and move it to In Progress.
2.  Sync and branch: git checkout main && git pull && git checkout -b feature/12-transfer-with-row-locking.
3.  Implement in small commits. Rebase on main regularly to avoid large conflicts.
4.  Open a draft pull request early so others can see progress.
5.  When acceptance criteria are met and CI is green, mark it ready for review and move the issue to In Review.
6.  Address review comments, then squash-merge. The issue closes automatically.
7.  Delete the branch and pick the next issue.
16.7 Avoiding Conflicts Between Contributors
• Own your area. Assign issues by package (for example one person on internal/database, one on internal/ledger, one on internal/auth) so that people rarely edit the same files.
• Agree interfaces first. When one feature depends on another, the first issue defines the Go interface or API contract and merges quickly, so others can build against it.
• Migrations are serialised. Use timestamped migration filenames and announce new migrations in the issue to avoid clashing schema changes.
• Shared files such as go.mod, the OpenAPI spec and docs/ are edited in small, quickly merged pull requests.
• Add a CODEOWNERS file so the right reviewers are requested automatically.
16.8 Definition of Done
An issue is done only when all of these are true:
• Acceptance criteria are met and demonstrated.
• Code is reviewed (two reviewers on money-path code) and merged to main.
• Unit tests, and integration or concurrency tests where relevant, pass in CI.
• Documentation, OpenAPI spec and migrations are updated.
• No new linter, gosec or govulncheck findings.
• The issue is closed through the pull request.
16.9 Starter Backlog
The IDs below are planning references. Put them in the issue title (for example [MB-007] Transfer with row locking) and let your tracker assign its own issue numbers. Create epics first, then the child issues.
Epic A: Persistence Foundation (Phase 0)
ID
Issue
Area
Depends on
MB-001
Implement internal/database connection pool and WithTx helper
database
none
MB-002
Initial migrations: customers, accounts, transactions, ledger_entries, audit_log
database
MB-001
MB-003
Rename Transctions to Transaction; clean domain models
cli / bank
none
MB-004
Config loading (DATABASE_URL, timeouts) and structured logging
infra
none
MB-005
CI pipeline: build, vet, staticcheck, gosec, govulncheck, tests
infra
none
Epic B: Core Money Operations (Phase 0)
ID
Issue
Area
Depends on
MB-006
Persist account creation
bank / database
MB-002
MB-007
Ledger package: double-entry posting engine
ledger
MB-002
MB-008
Deposit use case
bank
MB-007
MB-009
Withdrawal use case with balance check
bank
MB-007
MB-010
Transfer use case with ordered FOR UPDATE locking
bank
MB-007
MB-011
Concurrency test: parallel transfers conserve total money
test
MB-010
MB-012
CLI commands for deposit, withdraw, transfer and balance
cli
MB-008, MB-009, MB-010
Epic C: Secure API (Phase 1)
ID
Issue
Area
Depends on
MB-013
HTTP server skeleton, router, middleware (request ID, timeouts, recovery)
api
MB-004
MB-014
OpenAPI 3 specification and generated docs
api / docs
MB-013
MB-015
User model and password hashing (Argon2id)
auth
MB-002
MB-016
Login, access token and refresh token rotation
auth
MB-015
MB-017
RBAC middleware and object-level access checks
auth
MB-016
MB-018
MFA (TOTP) enrolment and verification
auth
MB-016
MB-019
Idempotency-Key middleware and storage
api
MB-013
MB-020
Audit log writer for all state changes
ledger / api
MB-002
MB-021
Dockerfile and docker-compose for local development
infra
MB-013
MB-022
Rate limiting and security headers
api
MB-013
Epic D: Banking Hardening (Phase 2)
ID
Issue
Area
Depends on
MB-023
Nightly reconciliation job (account balance vs ledger, global debit/credit check)
ledger
MB-007
MB-024
Reversals through compensating entries
ledger
MB-007
MB-025
Holds and available-balance tracking
ledger
MB-007
MB-026
Transaction limits and velocity rules
bank
MB-010
MB-027
Statements: query, PDF and CSV export
api
MB-014
MB-028
Transactional outbox and notification worker
infra
MB-002
MB-029
Maker-checker approval workflow
auth / bank
MB-017
Epic E: Compliance and Integrations (Phase 3)
ID
Issue
Area
Depends on
MB-030
KYC workflow and document storage
compliance
MB-017
MB-031
AML screening integration and alert queue
compliance
MB-030
MB-032
Field-level encryption for national ID and documents
security
MB-002
MB-033
Payment gateway integration with signed, idempotent webhooks
api
MB-019
MB-034
Compliance matrix and data-retention jobs
docs / compliance
MB-031
16.10 Suggested Assignment Model
With several contributors, split by area so that dependencies flow in one direction:
• Data owner: database, migrations, reconciliation (MB-001, 002, 023).
• Ledger and money owner: ledger, deposit, withdraw, transfer (MB-007 to 011, 024, 025).
• Auth and security owner: users, tokens, RBAC, MFA, encryption (MB-015 to 018, 032).
• API and platform owner: HTTP layer, OpenAPI, CI, Docker (MB-004, 005, 013, 014, 019, 021, 022).
• Compliance and docs owner: KYC, AML, compliance matrix, documentation (MB-030, 031, 034).
Hold a short weekly triage to move issues from Backlog to Ready, review blockers, and re-balance work.
