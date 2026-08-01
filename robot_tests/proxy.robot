*** Settings ***
Resource        resources.robot
Test Tags       acceptance


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
