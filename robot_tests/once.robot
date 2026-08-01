*** Settings ***
Library         Process
Library         supporting.py
Suite Setup     Prepare Once
Suite Teardown  Clean Once


*** Variables ***
${ONCE}         ${CURDIR}${/}..${/}bin${/}once
${NAMESPACE}    once-robot-test
${HOST}         once-robot.localhost
${IMAGE}        ghcr.io/basecamp/once-campfire:main


*** Test Cases ***
Goal: Show version and command help
    ${version}=    Run Once    version
    Should Match Regexp    ${version.stdout}    ^v[0-9].*
    ${help}=    Run Once    --help
    Should Contain    ${help.stdout}    Manage web applications from Docker images
    Should Contain    ${help.stdout}    accessory
    Should Contain    ${help.stdout}    exec

Goal: Configure and inspect the proxy
    ${configured}=    Run Once
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
    Should Contain    ${configured.stdout}    Proxy configured
    ${shown}=    Run Once    proxy    show
    Should Contain    ${shown.stdout}    bind=127.0.0.1
    Should Contain    ${shown.stdout}    http=${HTTP_PORT}

Goal: Deploy and list an application
    ${deployed}=    Run Once
    ...    deploy
    ...    ${IMAGE}
    ...    --host
    ...    ${HOST}
    ...    --disable-tls
    ...    --auto-update=false
    ...    timeout=5 minutes
    Should Contain    ${deployed.stdout}    Deploying ${HOST}
    ${listed}=    Run Once    list
    ${list_output}=    Strip Ansi    ${listed.stdout}
    Should Contain    ${list_output}    ${HOST} (running)

Goal: Stop start and execute in an application
    ${stopped}=    Run Once    stop    ${HOST}
    Should Contain    ${stopped.stdout}    Stopped ${HOST}
    ${stopped_list}=    Run Once    list
    ${stopped_output}=    Strip Ansi    ${stopped_list.stdout}
    Should Contain    ${stopped_output}    ${HOST} (stopped)
    ${started}=    Run Once    start    ${HOST}
    Should Contain    ${started.stdout}    Started ${HOST}
    ${executed}=    Run Once    exec    ${HOST}    echo    robot-exec
    Should Contain    ${executed.stdout}    robot-exec

Goal: Remove an application and its data
    ${removed}=    Run Once    remove    ${HOST}    --remove-data
    Should Contain    ${removed.stdout}    Removed ${HOST}
    ${listed}=    Run Once    list
    Should Not Contain    ${listed.stdout}    ${HOST}


*** Keywords ***
Prepare Once
    Clean Once
    ${http_port}=    Free Port
    ${https_port}=    Free Port
    ${metrics_port}=    Free Port
    Set Suite Variable    ${HTTP_PORT}    ${http_port}
    Set Suite Variable    ${HTTPS_PORT}    ${https_port}
    Set Suite Variable    ${METRICS_PORT}    ${metrics_port}

Clean Once
    Run Process
    ...    ${ONCE}
    ...    --namespace
    ...    ${NAMESPACE}
    ...    teardown
    ...    --remove-data
    ...    timeout=2 minutes
    ...    on_timeout=terminate

Run Once
    [Arguments]    @{arguments}    ${timeout}=2 minutes
    ${result}=    Run Process
    ...    ${ONCE}
    ...    --namespace
    ...    ${NAMESPACE}
    ...    @{arguments}
    ...    timeout=${timeout}
    ...    on_timeout=terminate
    Log    <b>STDOUT</b><pre>${result.stdout}</pre>    html=yes
    Log    <b>STDERR</b><pre>${result.stderr}</pre>    html=yes
    Should Be Equal As Integers
    ...    ${result.rc}
    ...    0
    ...    msg=STDOUT:\n${result.stdout}\nSTDERR:\n${result.stderr}
    RETURN    ${result}
