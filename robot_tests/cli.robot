*** Settings ***
Resource        resources.robot
Test Tags       acceptance


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
