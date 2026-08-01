*** Settings ***
Library     OperatingSystem
Library     Process
Library     supporting.py


*** Variables ***
${ONCE}                 ${CURDIR}${/}..${/}bin${/}once
${APP_IMAGE}            ghcr.io/basecamp/once-campfire:main
${ACCESSORY_IMAGE}      busybox:1.36.1
${NAMESPACE}            once-robot-uninitialized


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
