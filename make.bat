@echo off
SETLOCAL

set VERSION=1.0.0-rc.2

REM Set build number from git commit hash
for /f %%i in ('git rev-parse HEAD') do set BUILD=%%i

set LDFLAGS=-ldflags "-X main.version=%VERSION% -X main.build=%BUILD% -s -w -extldflags '-static'"

goto build

:build
    REM echo "=== Building Docker image ==="
    REM docker build -t rokett/ldap-query:latest -t rokett/ldap-query:v%VERSION% .
    REM docker push rokett/ldap-query:v%VERSION%
    REM docker push rokett/ldap-query:latest

    set GOARCH=amd64

    %GOPATH%\bin\packr2

    go build %LDFLAGS% .\cmd\ldap-queryd

    goto :clean

:clean
    %GOPATH%\bin\packr2 clean

    set VERSION=
    set BUILD=
    set LDFLAGS=
    set GOARCH=

    goto :EOF
