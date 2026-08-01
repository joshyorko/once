# RCC development and acceptance testing

Run these commands from the Once repository root on Linux. RCC builds the contained Python, Go, Invoke, and Robot Framework environment from `developer/toolkit.yaml` and `developer/setup.yaml`. RCC artifacts remain under `developer/tmp`; Once does not use a `.rcc` directory.

## Regular tasks

```zsh
rcc run -r developer/toolkit.yaml -t build
rcc run -r developer/toolkit.yaml -t test
rcc run -r developer/toolkit.yaml -t install
rcc run -r developer/toolkit.yaml -t robot
```

## Development tasks

```zsh
rcc run -r developer/toolkit.yaml --dev -t build
rcc run -r developer/toolkit.yaml --dev -t test
rcc run -r developer/toolkit.yaml --dev -t install
rcc run -r developer/toolkit.yaml --dev -t integration
rcc run -r developer/toolkit.yaml --dev -t robotSmoke
rcc run -r developer/toolkit.yaml --dev -t robot
```

## Evidence boundaries

- `test` is Docker-free and covers Go unit tests plus documentation contracts.
- `integration` is explicit and covers internal Docker and PTY contracts.
- `robotSmoke` now exercises tagged CLI and proxy capability cases. A passing run proves the shared harness can build the binary, allocate a unique namespace plus dynamic ports, and tear down without leaving `once-robot-*` containers or volumes behind.
- `robot` is wired for the broader `acceptance` tag, but comprehensive application, backup, accessory, and regression suites are not implemented yet.
- host-mutating commands are excluded from default Robot execution.
- a passing Robot run is not release or deployment proof.

## Artifacts and cleanup

Robot reports live below `developer/tmp/robot`. The shared harness allocates a unique namespace plus dynamic ports and reports leaked Docker resources during teardown.

## Improving this guide

Update this guide only from verified code, tests, command results, observed failures, or authoritative RCC behavior. Replace stale claims, preserve remaining uncertainty, and return the documentation receipt from `docs/skills/README.md`.
