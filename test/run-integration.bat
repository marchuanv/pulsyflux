@echo off
REM Integration test runner — runs Go and Node.js consumer verification tests.
REM Exit codes: 0 = all passed, 1 = one or more failed.

setlocal enabledelayedexpansion

set SCRIPT_DIR=%~dp0
set EXIT_CODE=0

echo === Integration Tests ===
echo.

REM --- Go consumer tests ---
echo --- Go consumer tests ---
pushd "%SCRIPT_DIR%go"
go test -v -count=1 ./...
if %ERRORLEVEL% neq 0 (
    echo Go consumer tests: FAIL
    set EXIT_CODE=1
) else (
    echo Go consumer tests: PASS
)
popd

echo.

REM --- Node.js consumer tests ---
echo --- Node.js consumer tests ---
node "%SCRIPT_DIR%nodejs\consumer-test.mjs"
if %ERRORLEVEL% neq 0 (
    echo Node.js consumer tests: FAIL
    set EXIT_CODE=1
) else (
    echo Node.js consumer tests: PASS
)

echo.
echo === Integration Tests Complete ===

if %EXIT_CODE% equ 0 (
    echo Result: ALL PASSED
) else (
    echo Result: SOME FAILED
)

exit /b %EXIT_CODE%
