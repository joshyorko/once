# Once Robot Acceptance Test Design

## Objective

Adopt the testing pattern used by `joshyorko/rcc`: RCC and Invoke prepare the contained environment and compiled binary, while capability-focused Robot Framework suites verify user-visible behavior through the real CLI.

The current five Robot tests are a smoke suite. The target is a comprehensive, container-safe acceptance layer without duplicating internal assertions already owned by Go tests.

## Testing Boundaries

### Go unit tests

Go unit tests own parsing, validation, command helpers, and isolated package logic. The `test` RCC task remains fast and Docker-free.

### Go integration tests

Go integration tests own internal Docker behavior that is clearer or more reliable to inspect from Go, including container labels, preserved settings, hooks, low-level Docker failures, and TUI behavior requiring a pseudo-terminal.

### Robot acceptance tests

Robot tests own observable workflows through the compiled `once` binary: output, exit status, state transitions, persistence, and cleanup. They must use an isolated namespace and must not inspect internal Go types.

Existing Go integration tests will remain until equivalent Robot coverage exists. Afterward, only direct user-visible duplicates may be removed; internal Docker assertions remain in Go.

## RCC and Invoke Task Surface

`developer/toolkit.yaml` remains a thin task manifest. `developer/call_invoke.py` remains a thin dispatcher. `tasks.py` owns task dependencies and commands.

The initial task surface will be:

- `test`: Go unit tests only; no Docker.
- `integration`: internal Go/Docker integration tests; explicit developer task.
- `robotSmoke`: fast Robot subset covering CLI availability and one application lifecycle.
- `robot`: complete container-safe Robot acceptance suite.
- `build` and `install`: retain their existing behavior.

Regular RCC tasks expose `build`, `test`, `install`, and `robot`. Developer RCC tasks expose `build`, `test`, `install`, `integration`, `robotSmoke`, and `robot`.

`robotSmoke` and `robot` build the local binary first. The complete `robot` task also runs Go unit tests before Robot, matching the RCC pattern where acceptance tests run against a locally proven binary.

A future `robotHost` task is reserved for commands that modify the workstation rather than the isolated Docker namespace. It will not be introduced until a disposable-host runner exists, and it will never be a dependency of `robot`, `test`, or `integration`.

## Suite Organization

The single `robot_tests/once.robot` file will be replaced by capability-focused suites:

- `robot_tests/__init__.robot`: top-level setup and teardown.
- `robot_tests/resources.robot`: shared process, assertion, namespace, and cleanup keywords.
- `robot_tests/cli.robot`: version, help, aliases, invalid commands, argument errors, and exit codes.
- `robot_tests/proxy.robot`: configure, show, reconfigure, and observable proxy state.
- `robot_tests/applications.robot`: deploy, list, update, start, stop, exec, remove, multiple applications, and hostname conflicts.
- `robot_tests/backup_restore.robot`: application-data backup and restore round trip.
- `robot_tests/accessories.robot`: accessory deploy, list, logs, stop, start, and remove.
- `robot_tests/teardown.robot`: complete namespace and data removal.
- `robot_tests/regressions.robot`: user-visible behavior preserved for previously fixed bugs.
- `robot_tests/supporting.py`: narrowly scoped helpers that Robot Framework cannot express clearly.

Robot tags define execution groups:

- `smoke`: CLI availability and one happy-path application lifecycle.
- `acceptance`: all container-safe workflows.
- `host`: reserved for future workstation-mutating workflows.

## Isolation and Cleanup

Each Robot invocation receives a unique Once namespace rather than sharing `once-robot-test`. It also receives dynamically allocated proxy ports and a unique temporary workspace under `developer/tmp`.

Top-level setup records the namespace, ports, and temporary paths. Top-level teardown runs regardless of test outcome and removes application containers, accessory containers, proxy containers, networks, and volumes created by that namespace.

Tests may share state within a capability suite when the sequence itself is under test. Separate capability suites must not depend on execution order or state created by another suite.

Cleanup must be idempotent. Failure to clean resources is reported as a test failure with the remaining resource names.

## Assertions and Failure Coverage

Shared keywords will support both successful and intentionally failing commands. Every command assertion records stdout, stderr, and the exit code in the Robot report.

Acceptance tests verify outcomes rather than implementation details. Required failure coverage includes:

- unknown commands and missing arguments;
- duplicate or conflicting hostnames;
- commands targeting missing applications or accessories;
- invalid proxy configuration;
- failed exec commands and propagation of their exit status;
- invalid backup or restore inputs;
- repeated remove and teardown behavior.

ANSI and terminal hyperlink sequences are normalized only for comparisons and reporting. Raw output remains available in the process result.

## Container-Safe Acceptance Coverage

The default `robot` task covers:

1. CLI help, version, aliases, validation, and exit statuses.
2. Proxy configuration and reconfiguration with dynamic ports.
3. Application deploy, list, update, stop, start, exec, and remove.
4. Multiple isolated applications and hostname collision handling.
5. Persistent data across stop, start, update, backup, removal, and restore.
6. Accessory lifecycle and logs using a deterministic lightweight fixture.
7. Full teardown and proof that no namespace resources remain.
8. User-visible regressions suited to black-box verification.

The existing first-run TUI test remains in Go integration because it already has pseudo-terminal-aware coverage. Moving it to Robot is outside this change.

## Host-Mutating Coverage

`self-update` and `background install` or `uninstall` modify the workstation outside the Docker namespace. Their argument validation and help remain covered by unit or container-safe CLI tests.

Actual host mutation will eventually belong only in `robotHost`, require explicit invocation, and verify prerequisites before changing the host. Neither the task nor host-mutating tests are part of the initial implementation because no disposable-host runner is currently defined.

## Artifacts

RCC keeps `artifactsDir: tmp`, resolving to `developer/tmp`. Robot Framework reports are written beneath `developer/tmp/robot`, with separate subdirectories for smoke, acceptance, and host runs.

Generated artifacts remain ignored by Git. A new run must not warn about stale RCC root artifacts, and test setup must clean only its own run-specific directory.

## Migration Sequence

1. Introduce shared Robot resources and unique-run isolation.
2. Move the existing smoke cases into capability suites and tag them `smoke`.
3. Add CLI failure and exit-code coverage.
4. Add application update, multi-app, collision, and persistence coverage.
5. Add backup and restore coverage.
6. Add accessory coverage using a deterministic fixture.
7. Add teardown resource-leak assertions.
8. Compare Robot scenarios with Go integration tests and remove only proven direct duplication.

## Acceptance Criteria

- `rcc run -r developer/toolkit.yaml --dev -t test` passes without starting Docker containers.
- `rcc run -r developer/toolkit.yaml --dev -t robotSmoke` passes and leaves no Docker resources.
- `rcc run -r developer/toolkit.yaml --dev -t robot` exercises all container-safe suites and leaves no Docker resources.
- A failing command can be asserted without the Robot helper forcing an immediate success-only failure.
- Robot reports are contained under `developer/tmp/robot`.
- No `.rcc` artifact directory is introduced.
- Existing internal Go integration coverage is retained unless a direct duplicate is demonstrated.
- Host-mutating commands never run through the default `test`, `integration`, or `robot` tasks.
