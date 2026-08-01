# Once Robot Acceptance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the five-case Robot smoke file with an RCC-driven, capability-focused acceptance suite for every container-safe Once CLI workflow.

**Architecture:** Keep `developer/toolkit.yaml` as the RCC manifest and `developer/call_invoke.py` as the dispatcher. Put task dependencies in `tasks.py`, common black-box process behavior in `robot_tests/resources.robot` and `robot_tests/supporting.py`, and user workflows in independent capability suites. Go unit tests remain Docker-free; Go integration tests retain internal Docker and PTY assertions.

**Tech Stack:** RCC 18.18.0, Invoke 2.2.0, Robot Framework 7.4.2, Python 3.10.15, Go 1.26.5, Docker, `ghcr.io/basecamp/once-campfire:main`, `busybox:1.36.1`.

## Global Constraints

- Run all developer workflows through `rcc run -r developer/toolkit.yaml`.
- Keep `artifactsDir: tmp`; do not introduce `.rcc`.
- Keep `test` Docker-free and make `integration` explicitly opt-in.
- Use a unique namespace, dynamic ports, and a run-specific directory for every Robot invocation.
- The default `robot` task must never execute `self-update` or `background install|uninstall`.
- Preserve existing Go integration tests until a direct duplicate is proven.
- Keep Robot reports below `developer/tmp/robot`.
- Keep reusable developer and operator guidance under `docs/skills/`, indexed by `docs/skills/README.md`.
- Every implementation lane must improve or propose an evidence-backed correction to canonical guidance and return the documentation receipt defined below.
- Do not add per-run diaries, cosmetic prose, or claims that planned behavior is implemented.
- Do not commit or push; repository instructions reserve those actions for the user.

---

## File Structure

- Modify `developer/toolkit.yaml`: expose `robotSmoke` in `devTasks` while retaining current regular tasks.
- Modify `AGENTS.md`: add the mandatory evidence-backed self-improvement contract.
- Modify `Makefile`: include documentation contract tests in the Docker-free `test` target.
- Create `docs/skills/README.md`: canonical guide index.
- Create `docs/skills/rcc-development.md`: RCC commands, test boundaries, artifacts, cleanup, and troubleshooting.
- Create `docs/docs_test.go`: reject broken links, undocumented RCC tasks, and overstated evidence.
- Modify `tasks.py`: add Robot tag selection, output directories, and unit-test/build prerequisites.
- Delete `robot_tests/once.robot`: replace the monolithic smoke suite after its cases are migrated in Task 4.
- Create `robot_tests/__init__.robot`: global setup and teardown.
- Create `robot_tests/resources.robot`: shared process, assertion, namespace, port, and cleanup keywords.
- Modify `robot_tests/supporting.py`: run IDs, ANSI normalization, and Docker resource discovery.
- Create `robot_tests/cli.robot`: CLI contract and exit-code coverage.
- Create `robot_tests/proxy.robot`: proxy configuration coverage.
- Create `robot_tests/applications.robot`: application lifecycle, update, collision, alias, and failure coverage.
- Create `robot_tests/backup_restore.robot`: persistent-data round trip.
- Create `robot_tests/accessories.robot`: accessory lifecycle and validation.
- Create `robot_tests/teardown.robot`: explicit full-namespace cleanup.
- Create `robot_tests/regressions.robot`: exec exit-code and output regressions.
- Retain `integration/docker_test.go` and `integration/ui_test.go`: internal state and PTY coverage.

## Per-Task Documentation Gate

Tasks 2 through 8 cannot complete until the implementer checks whether the observed code, command output, failure, cleanup behavior, or environment boundary changes `docs/skills/rcc-development.md`. Update the guide when the learning is durable; otherwise record why no canonical change is justified. Run `go test ./docs -count=1` after every guide edit.

Every task returns this receipt:

```text
Documentation improvement:
- Canonical file changed or proposed:
- Durable learning captured:
- Evidence:
- Stale or ambiguous guidance removed:
- Remaining uncertainty:
```

The receipt is an outcome summary, not a file to commit. The root integrator reconciles receipts into canonical guides before final completion.

### Task 1: Establish the evidence-backed self-improvement contract

**Files:**
- Modify: `AGENTS.md`
- Modify: `Makefile`
- Create: `docs/skills/README.md`
- Create: `docs/skills/rcc-development.md`
- Create: `docs/docs_test.go`

**Interfaces:**
- Produces the canonical RCC/Robot guide used by every later task.
- Produces Docker-free documentation contract tests run by `make test` and the RCC `test` task.
- Produces the mandatory per-task documentation receipt contract.

- [ ] **Step 1: Write documentation contract tests first**

Create `docs/docs_test.go` with these tests and the standard-library YAML-section scanner:

```go
package docs_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestSkillsIndexLinksExistingFiles(t *testing.T) {
	body, err := os.ReadFile("skills/README.md")
	if err != nil {
		t.Fatal(err)
	}
	links := regexp.MustCompile(`\[[^]]+\]\(([^):#]+\.md)\)`)
	for _, match := range links.FindAllStringSubmatch(string(body), -1) {
		if _, err := os.Stat(filepath.Join("skills", filepath.Clean(match[1]))); err != nil {
			t.Errorf("skills index link %q: %v", match[1], err)
		}
	}
}

func TestRCCGuideCoversEveryToolkitTask(t *testing.T) {
	toolkit, err := os.ReadFile("../developer/toolkit.yaml")
	if err != nil {
		t.Fatal(err)
	}
	guide, err := os.ReadFile("skills/rcc-development.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range taskNames(string(toolkit), "tasks") {
		command := "rcc run -r developer/toolkit.yaml -t " + task
		if !strings.Contains(string(guide), command) {
			t.Errorf("regular RCC task lacks documented command %q", command)
		}
	}
	for _, task := range taskNames(string(toolkit), "devTasks") {
		command := "rcc run -r developer/toolkit.yaml --dev -t " + task
		if !strings.Contains(string(guide), command) {
			t.Errorf("development RCC task lacks documented command %q", command)
		}
	}
}

func TestRCCGuideKeepsEvidenceBoundariesExplicit(t *testing.T) {
	body, err := os.ReadFile("skills/rcc-development.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, phrase := range []string{
		"`test` is Docker-free",
		"`integration` is explicit",
		"host-mutating commands are excluded",
		"a passing Robot run is not release or deployment proof",
	} {
		if !strings.Contains(string(body), phrase) {
			t.Errorf("RCC guide lacks evidence boundary %q", phrase)
		}
	}
}

func taskNames(document, section string) []string {
	header := section + ":"
	inSection := false
	name := regexp.MustCompile(`^  ([A-Za-z][A-Za-z0-9]*):$`)
	var names []string
	for _, line := range strings.Split(document, "\n") {
		if line == header {
			inSection = true
			continue
		}
		if inSection && line != "" && line[0] != ' ' {
			break
		}
		if inSection {
			if match := name.FindStringSubmatch(line); match != nil {
				names = append(names, match[1])
			}
		}
	}
	sort.Strings(names)
	return names
}
```

- [ ] **Step 2: Run the docs tests to verify RED**

Run:

```zsh
go test ./docs -count=1
```

Expected: FAIL because `docs/skills/README.md` and `docs/skills/rcc-development.md` do not exist.

- [ ] **Step 3: Add the repository self-improvement contract**

Add a `Mandatory Self-Improvement Contract` section before `Agent behaviour` in `AGENTS.md` with these requirements:

```text
Every implementation, review, debugging, reconnaissance, and verification task must leave Once's repository-local operational guidance measurably better in correctness, completeness, discoverability, determinism, testability, or recovery guidance.

- docs/skills/ is the canonical home for reusable operational knowledge; correct an existing guide before creating another.
- Document only behavior backed by code, tests, commands, observed failures, or authoritative upstream behavior.
- Replace stale or contradictory guidance when evidence changes; do not create per-run diaries or cosmetic prose.
- Mutating lanes update the canonical guide; read-only lanes propose an exact delta.
- Every lane returns the Documentation improvement receipt from docs/skills/README.md.
- Parent work cannot complete until receipts are reconciled and remaining uncertainty is explicit.
```

Preserve the existing prohibition on agent commits and pushes.

- [ ] **Step 4: Create the canonical skills index and RCC guide**

Create `docs/skills/README.md`:

````markdown
# Once operational skills

These guides contain reusable, evidence-backed developer and operator knowledge. Correct an existing guide before adding another; do not store per-run diaries here.

- [RCC development and acceptance testing](rcc-development.md)

Every mutating, debugging, review, or verification lane returns:

```text
Documentation improvement:
- Canonical file changed or proposed:
- Durable learning captured:
- Evidence:
- Stale or ambiguous guidance removed:
- Remaining uncertainty:
```
````

Create `docs/skills/rcc-development.md` with these sections and exact claims:

````markdown
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
- `robotSmoke` builds the binary and runs the fast black-box subset.
- `robot` runs unit tests, builds the binary, and exercises all container-safe CLI acceptance suites.
- host-mutating commands are excluded from default Robot execution.
- a passing Robot run is not release or deployment proof.

## Artifacts and cleanup

Robot reports live below `developer/tmp/robot`. Every Robot run uses a unique Once namespace, dynamic ports, and teardown that reports leaked containers, networks, or volumes. Generated artifacts are ignored and removed before handoff after reports are inspected.

## Improving this guide

Update this guide only from verified code, tests, command results, observed failures, or authoritative RCC behavior. Replace stale claims, preserve remaining uncertainty, and return the documentation receipt from `docs/skills/README.md`.
````

- [ ] **Step 5: Include docs contracts in the Docker-free test lane**

Change the `Makefile` test recipe to:

```make
test:
	go test ./internal/... ./docs
```

- [ ] **Step 6: Run the documentation and unit gates to verify GREEN**

Run:

```zsh
gofmt -w docs/docs_test.go
go test ./docs -count=1
rcc run -r developer/toolkit.yaml --dev -t test --silent
git diff --check
```

Expected: docs tests PASS, RCC `test` PASS without new Docker containers, and diff check PASS.

- [ ] **Step 7: Return the first documentation receipt**

```text
Documentation improvement:
- Canonical file changed or proposed: docs/skills/rcc-development.md
- Durable learning captured: RCC task commands, test-lane ownership, artifact location, cleanup contract, and evidence limits
- Evidence: developer/toolkit.yaml, tasks.py, Makefile, docs/docs_test.go, and the passing RCC test command
- Stale or ambiguous guidance removed: the five-case Robot suite is no longer described as comprehensive
- Remaining uncertainty: container-safe Robot scenarios remain unverified until Tasks 2 through 7 execute
```

### Task 2: Build the shared Robot harness and RCC task selectors

**Files:**
- Modify: `developer/toolkit.yaml`
- Modify: `tasks.py`
- Create: `robot_tests/__init__.robot`
- Create: `robot_tests/resources.robot`
- Modify: `robot_tests/supporting.py`

**Interfaces:**
- Produces Python keywords `make_run_id() -> str`, `free_port() -> int`, `strip_ansi(value: str) -> str`, and `once_resources(namespace: str) -> list[str]`.
- Produces Robot keywords `Prepare Once`, `Clean Once`, `Run Once Raw`, `Run Once`, `Run Once Expecting`, `Run Once In Namespace`, and `Configure Proxy`.
- Produces global Robot variables `${RUN_ID}`, `${NAMESPACE}`, `${RUN_DIR}`, `${HTTP_PORT}`, `${HTTPS_PORT}`, and `${METRICS_PORT}`.
- Produces Invoke tasks `robotSmoke` and `robot`.

- [ ] **Step 1: Add the failing RCC smoke-task declaration**

Add this entry under `devTasks` in `developer/toolkit.yaml` before defining its Invoke implementation:

```yaml
  robotSmoke:
    shell: python call_invoke.py robotSmoke
```

- [ ] **Step 2: Verify the new task fails for the expected reason**

Run:

```zsh
rcc run -r developer/toolkit.yaml --dev -t robotSmoke --silent
```

Expected: FAIL because Invoke cannot find `robotSmoke`; no Once test containers should exist.

- [ ] **Step 3: Extend the Python helper with unique-run and cleanup discovery functions**

Add these imports and functions to `robot_tests/supporting.py`, preserving `free_port` and `strip_ansi`:

```python
import subprocess
import uuid


def make_run_id():
    return uuid.uuid4().hex[:10]


def once_resources(namespace):
    commands = (
        ("docker", "ps", "-a", "--format", "{{.Names}}"),
        ("docker", "network", "ls", "--format", "{{.Name}}"),
        ("docker", "volume", "ls", "--format", "{{.Name}}"),
    )
    resources = []
    for command in commands:
        result = subprocess.run(command, check=True, capture_output=True, text=True)
        resources.extend(
            name
            for name in result.stdout.splitlines()
            if name == namespace or name.startswith(f"{namespace}-")
        )
    return sorted(set(resources))
```

- [ ] **Step 4: Create the top-level suite lifecycle**

Create `robot_tests/__init__.robot`:

```robotframework
*** Settings ***
Resource            resources.robot
Suite Setup         Prepare Once
Suite Teardown      Clean Once
```

Create `robot_tests/resources.robot` with these settings and variables:

```robotframework
*** Settings ***
Library     OperatingSystem
Library     Process
Library     supporting.py


*** Variables ***
${ONCE}                 ${CURDIR}${/}..${/}bin${/}once
${APP_IMAGE}            ghcr.io/basecamp/once-campfire:main
${ACCESSORY_IMAGE}      busybox:1.36.1
${NAMESPACE}            once-robot-uninitialized
```

Implement the shared keywords with this contract:

```robotframework
*** Keywords ***
Prepare Once
    ${run_id}=    Make Run Id
    Set Global Variable    ${RUN_ID}    ${run_id}
    Set Global Variable    ${NAMESPACE}    once-robot-${run_id}
    Set Global Variable    ${RUN_DIR}    ${OUTPUT DIR}${/}state
    Create Directory    ${RUN_DIR}
    ${http_port}=    Free Port
    ${https_port}=    Free Port
    ${metrics_port}=    Free Port
    Set Global Variable    ${HTTP_PORT}    ${http_port}
    Set Global Variable    ${HTTPS_PORT}    ${https_port}
    Set Global Variable    ${METRICS_PORT}    ${metrics_port}

Clean Once
    ${result}=    Run Process
    ...    ${ONCE}
    ...    --namespace
    ...    ${NAMESPACE}
    ...    teardown
    ...    --remove-data
    ...    timeout=3 minutes
    ...    on_timeout=terminate
    Log Process Result    ${result}
    ${resources}=    Once Resources    ${NAMESPACE}
    Should Be Empty    ${resources}    msg=Leaked Docker resources: ${resources}

Run Once Raw
    [Arguments]    @{arguments}    ${timeout}=2 minutes
    ${result}=    Run Process
    ...    ${ONCE}
    ...    --namespace
    ...    ${NAMESPACE}
    ...    @{arguments}
    ...    timeout=${timeout}
    ...    on_timeout=terminate
    Log Process Result    ${result}
    RETURN    ${result}

Run Once
    [Arguments]    @{arguments}    ${timeout}=2 minutes
    ${result}=    Run Once Raw    @{arguments}    timeout=${timeout}
    Should Be Equal As Integers
    ...    ${result.rc}
    ...    0
    ...    msg=STDOUT:\n${result.stdout}\nSTDERR:\n${result.stderr}
    RETURN    ${result}

Run Once Expecting
    [Arguments]    ${expected_rc}    @{arguments}    ${timeout}=2 minutes
    ${result}=    Run Once Raw    @{arguments}    timeout=${timeout}
    Should Be Equal As Integers
    ...    ${result.rc}
    ...    ${expected_rc}
    ...    msg=STDOUT:\n${result.stdout}\nSTDERR:\n${result.stderr}
    RETURN    ${result}

Run Once In Namespace
    [Arguments]    ${namespace}    @{arguments}    ${timeout}=2 minutes
    ${result}=    Run Process
    ...    ${ONCE}
    ...    --namespace
    ...    ${namespace}
    ...    @{arguments}
    ...    timeout=${timeout}
    ...    on_timeout=terminate
    Log Process Result    ${result}
    RETURN    ${result}

Log Process Result
    [Arguments]    ${result}
    Log    <b>STDOUT</b><pre>${result.stdout}</pre>    html=yes
    Log    <b>STDERR</b><pre>${result.stderr}</pre>    html=yes

Configure Proxy
    Run Once
    ...    proxy
    ...    configure
    ...    --bind
    ...    127.0.0.1
    ...    --http-port
    ...    ${HTTP_PORT}
    ...    --https-port
    ...    ${HTTPS_PORT}
    ...    --metrics-port
    ...    ${METRICS_PORT}
```

- [ ] **Step 5: Implement tagged Robot runners in Invoke**

Replace the current `robot` task in `tasks.py` with:

```python
@task(name="robotSmoke", pre=[build])
def robot_smoke(c):
    """Run the fast black-box acceptance subset."""
    _run_robot(c, "smoke", "developer/tmp/robot/smoke")


@task(pre=[test, build])
def robot(c):
    """Run all container-safe black-box acceptance tests."""
    _run_robot(c, "acceptance", "developer/tmp/robot/acceptance")


# Helpers

def _run_robot(c, tag, output_dir):
    c.run(f"python -m robot -L DEBUG --include {tag} -d {output_dir} robot_tests")
```

Keep the existing regular `robot` entry and add the `robotSmoke` development entry. Do not add `robotHost`.

- [ ] **Step 6: Validate harness syntax before capability tests exist**

Run:

```zsh
rcc task script -r developer/toolkit.yaml --silent -- python -m robot --dryrun robot_tests
```

Expected: PASS for resource parsing. If Robot reports that no matching tests exist, that is acceptable until Task 3 adds tagged cases.

- [ ] **Step 7: Review the task boundary**

Run:

```zsh
rcc run -r developer/toolkit.yaml --dev -t test --silent
git diff --check
```

Expected: Go unit tests PASS without new Docker containers; diff check PASS.

### Task 3: Migrate smoke coverage into CLI and proxy suites

**Files:**
- Create: `robot_tests/cli.robot`
- Create: `robot_tests/proxy.robot`
- Delete: `robot_tests/once.robot`

**Interfaces:**
- Consumes all shared variables and process keywords from `robot_tests/resources.robot`.
- Produces the complete `smoke` tag subset.

- [ ] **Step 1: Create CLI contract cases**

Create `robot_tests/cli.robot`:

```robotframework
*** Settings ***
Resource        resources.robot
Test Tags      acceptance


*** Test Cases ***
Goal: Show version and command help
    [Tags]    smoke
    ${version}=    Run Once    version
    Should Match Regexp    ${version.stdout}    ^v[0-9].*
    ${help}=    Run Once    --help
    Should Contain    ${help.stdout}    Manage web applications from Docker images
    Should Contain    ${help.stdout}    accessory
    Should Contain    ${help.stdout}    backup
    Should Contain    ${help.stdout}    restore
    Should Contain    ${help.stdout}    update

Goal: Reject an unknown command
    ${result}=    Run Once Expecting    1    not-a-command
    Should Contain    ${result.stderr}    unknown command

Goal: Require deploy arguments
    ${result}=    Run Once Expecting    1    deploy
    Should Contain    ${result.stderr}    accepts 1 arg

Goal: Require exec command arguments
    ${result}=    Run Once Expecting    1    exec    missing.localhost
    Should Contain    ${result.stderr}    requires at least 2 arg

Goal: Expose list and remove aliases
    ${list_help}=    Run Once    ls    --help
    Should Contain    ${list_help.stdout}    List installed applications
    ${remove_help}=    Run Once    rm    --help
    Should Contain    ${remove_help.stdout}    Remove an application

Goal: Show host-mutating commands without executing them
    ${self_update}=    Run Once    self-update    --help
    Should Contain    ${self_update.stdout}    Update once to the latest version
    ${background}=    Run Once    background    --help
    Should Contain    ${background.stdout}    install
    Should Contain    ${background.stdout}    uninstall
```

- [ ] **Step 2: Verify CLI tests fail before the old suite is removed only if shared setup is incomplete**

Run:

```zsh
rcc task script -r developer/toolkit.yaml --silent -- python -m robot -d developer/tmp/robot/cli --suite Cli robot_tests
```

Expected: PASS. Any output mismatch must be corrected against the actual Cobra output, not weakened to a generic success assertion.

- [ ] **Step 3: Create proxy behavior cases**

Create `robot_tests/proxy.robot`:

```robotframework
*** Settings ***
Resource        resources.robot
Test Tags      acceptance


*** Test Cases ***
Goal: Show default proxy settings
    ${shown}=    Run Once    proxy    show
    Should Contain    ${shown.stdout}    bind=0.0.0.0
    Should Contain    ${shown.stdout}    http=80
    Should Contain    ${shown.stdout}    https=443

Goal: Configure and inspect the proxy
    [Tags]    smoke
    Configure Proxy
    ${shown}=    Run Once    proxy    show
    Should Contain    ${shown.stdout}    bind=127.0.0.1
    Should Contain    ${shown.stdout}    http=${HTTP_PORT}
    Should Contain    ${shown.stdout}    https=${HTTPS_PORT}
    Should Contain    ${shown.stdout}    metrics=${METRICS_PORT}

Goal: Reconfigure the proxy
    ${replacement}=    Free Port
    Run Once    proxy    configure    --bind    127.0.0.1    --http-port    ${replacement}    --https-port    ${HTTPS_PORT}    --metrics-port    ${METRICS_PORT}
    ${shown}=    Run Once    proxy    show
    Should Contain    ${shown.stdout}    http=${replacement}
```

- [ ] **Step 4: Keep the monolithic suite until its application cases move**

Do not delete `robot_tests/once.robot` yet. Its version/help and proxy cases are now duplicated temporarily; its deploy/list, lifecycle, exec, and remove cases move in Task 4.

- [ ] **Step 5: Run the new smoke selector**

Run:

```zsh
rcc run -r developer/toolkit.yaml --dev -t robotSmoke --silent
```

Expected: PASS; the report is under `developer/tmp/robot/smoke`; teardown reports no leaked namespace resources.

### Task 4: Add application lifecycle, update, collision, alias, and failure scenarios

**Files:**
- Create: `robot_tests/applications.robot`

**Interfaces:**
- Consumes `${APP_IMAGE}`, `${RUN_ID}`, `Configure Proxy`, `Run Once`, and `Run Once Expecting`.
- Uses suite-local hosts `${PRIMARY_HOST}`, `${UPDATED_HOST}`, and `${SECONDARY_HOST}`.

- [ ] **Step 1: Create independent application-suite setup and cleanup**

Start `robot_tests/applications.robot` with:

```robotframework
*** Settings ***
Resource            resources.robot
Suite Setup         Prepare Application Suite
Suite Teardown      Clean Application Suite
Test Tags           acceptance


*** Keywords ***
Prepare Application Suite
    Configure Proxy
    Set Suite Variable    ${PRIMARY_HOST}    app-${RUN_ID}.localhost
    Set Suite Variable    ${UPDATED_HOST}    updated-${RUN_ID}.localhost
    Set Suite Variable    ${SECONDARY_HOST}    second-${RUN_ID}.localhost

Clean Application Suite
    Run Once Raw    remove    ${PRIMARY_HOST}    --remove-data
    Run Once Raw    remove    ${UPDATED_HOST}    --remove-data
    Run Once Raw    remove    ${SECONDARY_HOST}    --remove-data
```

- [ ] **Step 2: Add the smoke lifecycle and verify it fails if the binary contract changed**

Add:

```robotframework
*** Test Cases ***
Goal: Deploy and list an application
    [Tags]    smoke
    ${deployed}=    Run Once    deploy    ${APP_IMAGE}    --host    ${PRIMARY_HOST}    --disable-tls    --auto-update=false    timeout=5 minutes
    Should Contain    ${deployed.stdout}    Deploying ${PRIMARY_HOST}
    ${listed}=    Run Once    list
    ${output}=    Strip Ansi    ${listed.stdout}
    Should Contain    ${output}    ${PRIMARY_HOST} (running)

Goal: Stop start and execute in an application
    [Tags]    smoke
    ${stopped}=    Run Once    stop    ${PRIMARY_HOST}
    Should Contain    ${stopped.stdout}    Stopped ${PRIMARY_HOST}
    ${listed}=    Run Once    ls
    ${output}=    Strip Ansi    ${listed.stdout}
    Should Contain    ${output}    ${PRIMARY_HOST} (stopped)
    ${not_running}=    Run Once Expecting    1    exec    ${PRIMARY_HOST}    echo    should-not-run
    Should Contain    ${not_running.stderr}    executing command in application
    ${started}=    Run Once    start    ${PRIMARY_HOST}
    Should Contain    ${started.stdout}    Started ${PRIMARY_HOST}
    ${executed}=    Run Once    exec    ${PRIMARY_HOST}    echo    robot-exec
    Should Contain    ${executed.stdout}    robot-exec
```

Run:

```zsh
rcc task script -r developer/toolkit.yaml --silent -- python -m robot -d developer/tmp/robot/applications --suite Applications robot_tests
```

Expected: PASS through the built CLI.

- [ ] **Step 3: Add update and hostname-transition coverage**

Add:

```robotframework
Goal: Update application settings and hostname
    ${updated}=    Run Once    update    ${PRIMARY_HOST}    --host    ${UPDATED_HOST}    --env    ROBOT_UPDATE=applied    --auto-update=false    timeout=5 minutes
    Should Contain    ${updated.stdout}    Updating ${PRIMARY_HOST}
    ${listed}=    Run Once    list
    ${output}=    Strip Ansi    ${listed.stdout}
    Should Not Contain    ${output}    ${PRIMARY_HOST}
    Should Contain    ${output}    ${UPDATED_HOST} (running)
    ${environment}=    Run Once    exec    ${UPDATED_HOST}    printenv    ROBOT_UPDATE
    Should Contain    ${environment.stdout}    applied
```

- [ ] **Step 4: Add multi-app and collision coverage**

Add:

```robotframework
Goal: Deploy multiple applications and reject host collisions
    Run Once    deploy    ${APP_IMAGE}    --host    ${SECONDARY_HOST}    --disable-tls    --auto-update=false    timeout=5 minutes
    ${collision}=    Run Once Expecting    1    deploy    ${APP_IMAGE}    --host    ${SECONDARY_HOST}    --disable-tls    --auto-update=false    timeout=5 minutes
    Should Contain    ${collision.stderr}    hostname is already in use
    ${update_collision}=    Run Once Expecting    1    update    ${SECONDARY_HOST}    --host    ${UPDATED_HOST}    timeout=5 minutes
    Should Contain    ${update_collision.stderr}    hostname is already in use
```

- [ ] **Step 5: Add missing-target and alias removal coverage**

Add:

```robotframework
Goal: Report missing application targets
    ${stop}=    Run Once Expecting    1    stop    absent-${RUN_ID}.localhost
    Should Contain    ${stop.stderr}    no application found
    ${remove}=    Run Once Expecting    1    remove    absent-${RUN_ID}.localhost
    Should Contain    ${remove.stderr}    no application found

Goal: Remove applications through both command names
    ${removed}=    Run Once    rm    ${SECONDARY_HOST}    --remove-data
    Should Contain    ${removed.stdout}    Removed ${SECONDARY_HOST}
    ${removed_updated}=    Run Once    remove    ${UPDATED_HOST}    --remove-data
    Should Contain    ${removed_updated.stdout}    Removed ${UPDATED_HOST}
    ${listed}=    Run Once    list
    Should Not Contain    ${listed.stdout}    ${SECONDARY_HOST}
    Should Not Contain    ${listed.stdout}    ${UPDATED_HOST}
```

- [ ] **Step 6: Run the application suite twice to prove cleanup and repeatability**

Delete `robot_tests/once.robot` now that all five original cases have been migrated.

Run twice:

```zsh
rcc task script -r developer/toolkit.yaml --silent -- python -m robot -d developer/tmp/robot/applications --suite Applications robot_tests
```

Expected both times: PASS; no resources with either generated namespace remain.

### Task 5: Add persistent-data backup and restore acceptance

**Files:**
- Create: `robot_tests/backup_restore.robot`

**Interfaces:**
- Consumes `${APP_IMAGE}`, `${RUN_DIR}`, `${RUN_ID}`, `Configure Proxy`, `Run Once`, and `Run Once Expecting`.
- Produces `${BACKUP_FILE}` only beneath the run-specific Robot output directory.

- [ ] **Step 1: Create the suite setup and a real storage marker**

Create `robot_tests/backup_restore.robot`:

```robotframework
*** Settings ***
Resource            resources.robot
Suite Setup         Prepare Backup Suite
Suite Teardown      Clean Backup Suite
Test Tags           acceptance


*** Keywords ***
Prepare Backup Suite
    Configure Proxy
    Set Suite Variable    ${BACKUP_HOST}    backup-${RUN_ID}.localhost
    Set Suite Variable    ${BACKUP_FILE}    ${RUN_DIR}${/}once-backup.tar.gz
    Run Once    deploy    ${APP_IMAGE}    --host    ${BACKUP_HOST}    --disable-tls    --auto-update=false    timeout=5 minutes
    Run Once    exec    ${BACKUP_HOST}    sh    -c    echo robot-persistent-data > /storage/robot-marker

Clean Backup Suite
    Run Once Raw    remove    ${BACKUP_HOST}    --remove-data
    Remove File    ${BACKUP_FILE}
```

- [ ] **Step 2: Add the backup, destructive removal, and restore round trip**

Add:

```robotframework
*** Test Cases ***
Goal: Backup and restore persistent application data
    Run Once    stop    ${BACKUP_HOST}
    ${backed_up}=    Run Once    backup    ${BACKUP_HOST}    ${BACKUP_FILE}    timeout=5 minutes
    Should Contain    ${backed_up.stdout}    Backed up ${BACKUP_HOST}
    File Should Exist    ${BACKUP_FILE}
    Run Once    remove    ${BACKUP_HOST}    --remove-data
    ${restored}=    Run Once    restore    ${BACKUP_FILE}    timeout=5 minutes
    Should Contain    ${restored.stdout}    Restored
    ${marker}=    Run Once    exec    ${BACKUP_HOST}    cat    /storage/robot-marker
    Should Contain    ${marker.stdout}    robot-persistent-data
```

- [ ] **Step 3: Add restore conflict and invalid-input failures**

Add after the round trip:

```robotframework
Goal: Reject restore when the hostname already exists
    ${result}=    Run Once Expecting    1    restore    ${BACKUP_FILE}    timeout=5 minutes
    Should Contain    ${result.stderr}    hostname is already in use

Goal: Reject a missing backup file
    ${result}=    Run Once Expecting    1    restore    ${RUN_DIR}${/}missing-backup.tar.gz
    Should Contain    ${result.stderr}    opening backup file
```

- [ ] **Step 4: Run the focused backup suite**

Run:

```zsh
rcc task script -r developer/toolkit.yaml --silent -- python -m robot -d developer/tmp/robot/backup --suite "Backup Restore" robot_tests
```

Expected: PASS; restored `cat /storage/robot-marker` prints `robot-persistent-data`.

### Task 6: Add accessory lifecycle and validation acceptance

**Files:**
- Create: `robot_tests/accessories.robot`

**Interfaces:**
- Consumes `${ACCESSORY_IMAGE}`, `${RUN_ID}`, `Run Once`, and `Run Once Expecting`.
- Uses a shared accessory named `helper-${RUN_ID}` with restart policy `no`.

- [ ] **Step 1: Create a deterministic accessory fixture**

Create `robot_tests/accessories.robot`:

```robotframework
*** Settings ***
Resource            resources.robot
Suite Setup         Prepare Accessory Suite
Suite Teardown      Clean Accessory Suite
Test Tags           acceptance


*** Keywords ***
Prepare Accessory Suite
    Set Suite Variable    ${ACCESSORY_NAME}    helper-${RUN_ID}

Clean Accessory Suite
    Run Once Raw    accessory    remove    ${ACCESSORY_NAME}    --remove-data
```

- [ ] **Step 2: Add deploy, list, log, stop, start, and remove behavior**

Add:

```robotframework
*** Test Cases ***
Goal: Manage a shared accessory lifecycle
    ${deployed}=    Run Once
    ...    accessory
    ...    deploy
    ...    --name
    ...    ${ACCESSORY_NAME}
    ...    --image
    ...    ${ACCESSORY_IMAGE}
    ...    --restart
    ...    no
    ...    --cmd
    ...    sh
    ...    --cmd
    ...    -c
    ...    --cmd
    ...    echo robot-accessory-ready; exec sleep 600
    ...    timeout=3 minutes
    Should Contain    ${deployed.stdout}    Deployed ${ACCESSORY_NAME}
    ${listed}=    Run Once    accessory    list
    Should Contain    ${listed.stdout}    ${ACCESSORY_NAME}
    ${logs}=    Run Once    accessory    logs    ${ACCESSORY_NAME}    --lines    20
    Should Contain    ${logs.stdout}    robot-accessory-ready
    Run Once    accessory    stop    ${ACCESSORY_NAME}
    Run Once    accessory    start    ${ACCESSORY_NAME}
    Run Once    accessory    remove    ${ACCESSORY_NAME}    --remove-data
    ${listed_after}=    Run Once    accessory    list
    Should Not Contain    ${listed_after.stdout}    ${ACCESSORY_NAME}
```

- [ ] **Step 3: Add validation and missing-target failures**

Add:

```robotframework
Goal: Validate accessory deployment arguments
    ${missing_name}=    Run Once Expecting    1    accessory    deploy    --image    ${ACCESSORY_IMAGE}
    Should Contain    ${missing_name.stderr}    accessory name is required
    ${unknown_template}=    Run Once Expecting    1    accessory    deploy    --name    bad-${RUN_ID}    --template    not-a-template
    Should Contain    ${unknown_template.stderr}    unknown template
    ${invalid_env}=    Run Once Expecting    1    accessory    deploy    --name    bad-${RUN_ID}    --image    ${ACCESSORY_IMAGE}    --env    NOT_AN_ASSIGNMENT
    Should Contain    ${invalid_env.stderr}    invalid env

Goal: Report a missing accessory
    ${result}=    Run Once Expecting    1    accessory    stop    absent-${RUN_ID}
    Should Contain    ${result.stderr}    accessory "absent-${RUN_ID}" not found
```

- [ ] **Step 4: Run the focused accessory suite twice**

Run twice:

```zsh
rcc task script -r developer/toolkit.yaml --silent -- python -m robot -d developer/tmp/robot/accessories --suite Accessories robot_tests
```

Expected both times: PASS; no `helper-*` container or volume remains.

### Task 7: Add teardown and user-visible regression suites

**Files:**
- Create: `robot_tests/teardown.robot`
- Create: `robot_tests/regressions.robot`

**Interfaces:**
- Consumes `Run Once`, `Run Once Expecting`, `Run Once In Namespace`, `Once Resources`, and `Strip Ansi`.
- Uses a secondary namespace so explicit teardown tests cannot disrupt other suites.

- [ ] **Step 1: Add isolated full-teardown coverage**

Create `robot_tests/teardown.robot`:

```robotframework
*** Settings ***
Resource        resources.robot
Test Tags      acceptance


*** Test Cases ***
Goal: Teardown removes every resource in its namespace
    ${namespace}=    Set Variable    once-teardown-${RUN_ID}
    ${host}=    Set Variable    teardown-${RUN_ID}.localhost
    ${configured}=    Run Once In Namespace    ${namespace}    proxy    configure    --bind    127.0.0.1    --http-port    ${HTTP_PORT}    --https-port    ${HTTPS_PORT}    --metrics-port    ${METRICS_PORT}
    Should Be Equal As Integers    ${configured.rc}    0
    ${deployed}=    Run Once In Namespace    ${namespace}    deploy    ${APP_IMAGE}    --host    ${host}    --disable-tls    --auto-update=false    timeout=5 minutes
    Should Be Equal As Integers    ${deployed.rc}    0
    ${removed}=    Run Once In Namespace    ${namespace}    teardown    --remove-data    timeout=3 minutes
    Should Be Equal As Integers    ${removed.rc}    0
    Should Contain    ${removed.stdout}    Teardown complete
    ${resources}=    Once Resources    ${namespace}
    Should Be Empty    ${resources}

Goal: Repeated teardown is idempotent
    ${namespace}=    Set Variable    once-empty-${RUN_ID}
    ${first}=    Run Once In Namespace    ${namespace}    teardown    --remove-data
    Should Be Equal As Integers    ${first.rc}    0
    ${second}=    Run Once In Namespace    ${namespace}    teardown    --remove-data
    Should Be Equal As Integers    ${second.rc}    0
```

- [ ] **Step 2: Add exec exit-code propagation and terminal-output regressions**

Create `robot_tests/regressions.robot`:

```robotframework
*** Settings ***
Resource            resources.robot
Suite Setup         Prepare Regression Suite
Suite Teardown      Clean Regression Suite
Test Tags           acceptance


*** Keywords ***
Prepare Regression Suite
    Configure Proxy
    Set Suite Variable    ${REGRESSION_HOST}    regression-${RUN_ID}.localhost
    Run Once    deploy    ${APP_IMAGE}    --host    ${REGRESSION_HOST}    --disable-tls    --auto-update=false    timeout=5 minutes

Clean Regression Suite
    Run Once Raw    remove    ${REGRESSION_HOST}    --remove-data


*** Test Cases ***
Goal: Preserve application command exit status
    ${result}=    Run Once Expecting    23    exec    ${REGRESSION_HOST}    sh    -c    exit 23
    Should Be Empty    ${result.stdout}

Goal: Render list output without leaking terminal control sequences into comparisons
    ${listed}=    Run Once    list
    ${output}=    Strip Ansi    ${listed.stdout}
    Should Contain    ${output}    ${REGRESSION_HOST} (running)
    Should Not Contain    ${output}    \x1b
```

- [ ] **Step 3: Run both focused suites**

Run:

```zsh
rcc task script -r developer/toolkit.yaml --silent -- python -m robot -d developer/tmp/robot/teardown --suite Teardown robot_tests
rcc task script -r developer/toolkit.yaml --silent -- python -m robot -d developer/tmp/robot/regressions --suite Regressions robot_tests
```

Expected: PASS; the exit-code case returns exactly 23 and both namespaces are empty afterward.

### Task 8: Run the complete acceptance ladder and audit overlap

**Files:**
- Review: `robot_tests/*.robot`
- Review: `integration/docker_test.go`
- Review: `integration/ui_test.go`
- Review and reconcile: `docs/skills/rcc-development.md`
- Test: `docs/docs_test.go`
- Modify only if proven redundant: `integration/docker_test.go`

**Interfaces:**
- Consumes every suite and task created above.
- Produces a verified split between unit, integration, smoke, and complete Robot lanes plus a reconciled canonical guide and documentation receipt.

- [ ] **Step 1: Validate Robot syntax for the entire directory**

Run:

```zsh
rcc task script -r developer/toolkit.yaml --silent -- python -m robot --dryrun robot_tests
```

Expected: PASS with no missing keywords, duplicate test names, or variable-resolution errors.

- [ ] **Step 2: Prove the Docker-free unit lane**

Capture the names of running containers before and after:

```zsh
docker ps --format '{{.Names}}'
rcc run -r developer/toolkit.yaml --dev -t test --silent
docker ps --format '{{.Names}}'
```

Expected: unit tests PASS and the before/after container-name sets are identical.

- [ ] **Step 3: Run the smoke lane twice**

Run twice:

```zsh
rcc run -r developer/toolkit.yaml --dev -t robotSmoke --silent
```

Expected: both runs PASS; reports are under `developer/tmp/robot/smoke`; no generated namespace resources remain.

- [ ] **Step 4: Run the complete container-safe Robot lane**

Run:

```zsh
rcc run -r developer/toolkit.yaml --dev -t robot --silent
```

Expected: every `acceptance` test PASS; reports are under `developer/tmp/robot/acceptance`; no generated namespace resources remain.

- [ ] **Step 5: Run the existing internal Docker integration lane separately**

Run:

```zsh
rcc run -r developer/toolkit.yaml --dev -t integration --silent
```

Expected: PASS. This run is explicit and is not triggered by `test` or `robotSmoke`.

- [ ] **Step 6: Audit rather than automatically delete Go integration cases**

Compare Robot scenarios with these Go tests:

```text
TestStartStop
TestExec
TestBackup
TestRestore
TestRemoveApplication
TestRemoveApplicationWithData
TestUpdateChangeHost
TestUpdateHostCollision
```

Retain each Go test if it asserts internal Docker state, volume identity, container replacement, namespace restoration, hooks, or label persistence. Remove a Go test only when every assertion is both user-visible and already proven by Robot. Do not modify `integration/ui_test.go`.

- [ ] **Step 7: Reconcile documentation receipts and reject stale guidance**

Review every Task 2 through 7 receipt. Update `docs/skills/rcc-development.md` only with durable findings backed by the completed commands and reports. Remove claims disproven by observed behavior, and list any skipped image pull, Docker condition, platform state, or host-mutating path under remaining uncertainty rather than presenting it as passed.

Run:

```zsh
go test ./docs -count=1
rcc run -r developer/toolkit.yaml --dev -t test --silent
```

Expected: docs contracts PASS, every current RCC task has an exact documented command, and the guide continues to distinguish source/unit proof, Docker integration proof, Robot acceptance proof, and release/deployment proof.

- [ ] **Step 8: Perform final leak, artifact, and diff checks**

Run:

```zsh
docker ps -a --format '{{.Names}}'
docker network ls --format '{{.Name}}'
docker volume ls --format '{{.Name}}'
git diff --check
git status --short --branch
```

Expected: no names beginning with `once-robot-`, `once-teardown-`, or `once-empty-`; no `.rcc` path; diff check PASS. Preserve `developer/tmp` only long enough for report inspection, then remove generated artifacts before handoff.

Return the integrated receipt:

```text
Documentation improvement:
- Canonical file changed or proposed: docs/skills/rcc-development.md
- Durable learning captured: the verified RCC task matrix, actual Robot suite boundaries, cleanup behavior, and any confirmed troubleshooting guidance
- Evidence: named unit, docs, smoke, acceptance, integration, and Docker leak-check commands with their separate results
- Stale or ambiguous guidance removed: every claim contradicted by the completed run or superseded task name
- Remaining uncertainty: every skipped host-mutating, platform-specific, registry, release, or deployment state
```

Because repository instructions prohibit agent commits and pushes, finish with an unstaged or staged reviewable worktree as directed by the user. Report each verification lane separately; do not collapse unit, integration, Robot, and cleanup status into one result.
