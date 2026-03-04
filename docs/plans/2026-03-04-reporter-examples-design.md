# Reporter Example CLI Applications

## Goal

Add four scenario-based example CLI applications that demonstrate idiomatic usage of the `reporter` package. Serve as both integration reference for library consumers and quick runnable demos for repo browsers.

## Layout

```
_examples/
  fileprocess/main.go
  deploy/main.go
  healthcheck/main.go
  migration/main.go
```

Each example is a standalone `package main` under the `_examples/` directory. The `_` prefix excludes them from `go build ./...`. No shared helper packages between examples. Stdlib only.

## Shared Pattern

Every example follows the same structure:

1. Parse `--format` flag (`text` | `json` | `noop`, default `text`)
2. Construct the appropriate `reporter.Reporter`
3. Run the scenario using only the `Reporter` interface
4. Use `IsJSON()` to conditionally print non-reporter output (demonstrates the discriminator)

Each file is self-contained, under ~80 lines. Uses `time.Sleep` (~200ms) between steps for visible pacing.

## Scenarios

### 1. fileprocess — Batch File Processing

Simulates processing 5 files. Exercises: `Step` for each file, `Progress` as percentage, `Message` for details, `Complete` with summary.

**Text output:**
```
Step 1/5: Scanning document.pdf...
  Found 42 pages
Step 2/5: Scanning photo.jpg...
  ...
=================================================================
Processed 5 files (3 succeeded, 2 skipped)
=================================================================
```

`IsJSON()` check: if not JSON, prints a hint about `--format=json` at the end.

### 2. deploy — Multi-Step Deploy Pipeline

Simulates a 4-step deploy: validate config, build, run tests, deploy. Exercises: `Step`, `Warning` (deprecated config key), `Error` (non-fatal test issue), `Complete` with structured details (version, environment).

**Text output:**
```
Step 1/4: Validating configuration...
  Config loaded from deploy.yaml
Warning: deprecated key 'legacy_mode' in config

Step 2/4: Building application...
  Compiled 12 packages

Step 3/4: Running tests...
  47 passed, 1 skipped
Warning: test coverage below 80%

Step 4/4: Deploying to staging...
  Deployed to staging-01.example.com

=================================================================
Deploy complete: v1.2.3 → staging
=================================================================
```

### 3. healthcheck — Service Health Checks

Simulates checking 4 services (database, cache, API gateway, message queue). Exercises: `Step` per service, `Message`/`MessagePlain` for status, `Warning` for degraded, `Error` for down, `Complete` with overall status.

**Text output:**
```
Step 1/4: Checking database...
  PostgreSQL: connected (latency 12ms)

Step 2/4: Checking cache...
  Redis: connected (latency 2ms)

Step 3/4: Checking API gateway...
Warning: API gateway response time degraded (850ms)

Step 4/4: Checking message queue...
Error: message queue unreachable: connection refused

=================================================================
Health check: 2 healthy, 1 degraded, 1 down
=================================================================
```

### 4. migration — Data Migration

Simulates migrating 100 records in 5 batches of 20. Exercises: `Step` for phases (validate, migrate, verify), `Progress` with percentage updates, `Warning` for skipped records, `Complete` with stats.

**Text output:**
```
Step 1/3: Validating source data...
  Found 100 records to migrate

Step 2/3: Migrating records...
  20% — batch 1 of 5 complete
  40% — batch 2 of 5 complete
Warning: 3 records skipped (missing required field)
  60% — batch 3 of 5 complete
  80% — batch 4 of 5 complete
  100% — batch 5 of 5 complete

Step 3/3: Verifying migration...
  97 records verified

=================================================================
Migration complete: 97 migrated, 3 skipped, 0 failed
=================================================================
```

## Coverage

Collectively, the four examples exercise every method on the `Reporter` interface:

| Method        | fileprocess | deploy | healthcheck | migration |
|---------------|:-----------:|:------:|:-----------:|:---------:|
| Step          | x           | x      | x           | x         |
| Progress      | x           |        |             | x         |
| Message       | x           | x      | x           | x         |
| MessagePlain  |             |        | x           |           |
| Warning       |             | x      | x           | x         |
| Error         |             |        | x           |           |
| Complete      | x           | x      | x           | x         |
| IsJSON        | x           |        |             |           |

## Non-Goals

- No tests for examples (they are the tests — run them manually)
- No external dependencies
- No shared helper packages
