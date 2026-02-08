@echo off
echo Building messagebus library for Node.js...
set PATH=C:\msys64\mingw64\bin;%PATH%
set CC=gcc
set CGO_ENABLED=1
set GOCACHE=%CD%\build\.cache
go build -buildmode=c-shared -o build\messagebus_lib.dll messagebus_lib.go

if %errorlevel% neq 0 (
    echo Build failed!
    exit /b 1
)

echo Build successful!
echo Generated: build/messagebus_lib.dll and messagebus_lib.h
