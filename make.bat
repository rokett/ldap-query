@echo off
SETLOCAL

set VERSION=1.2.2

REM Set build number from git commit hash
for /f %%i in ('git rev-parse HEAD') do set BUILD=%%i

set LDFLAGS=-ldflags "-X main.version=%VERSION% -X main.build=%BUILD% -s -w -extldflags '-static'"

goto build

:build
    echo "=== Building Docker image ==="
    docker build -t rokett/ldap-query:latest -t rokett/ldap-query:v%VERSION% .
    docker push rokett/ldap-query:v%VERSION%
    docker push rokett/ldap-query:latest

    set GOARCH=amd64

    go build -mod=vendor %LDFLAGS% .\cmd\ldap-queryd

    goto :clean

:clean
    set VERSION=
    set BUILD=
    set LDFLAGS=
    set GOARCH=

    goto :EOF
