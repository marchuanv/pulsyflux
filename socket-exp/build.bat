@echo off
echo Building socket library for Node.js...
set PATH=C:\msys64\mingw64\bin;%PATH%
set CC=gcc
set CGO_ENABLED=1
set GOCACHE=%CD%\build\.cache
set GOWORK=off
cd /d e:\github\pulsyflux
go build -buildmode=c-shared -o socket-exp/build/socket_lib.dll ./socket-exp/socket_lib.go

if %errorlevel% neq 0 (
    echo Build failed!
    exit /b 1
)

cd socket-exp
copy build\socket_lib.dll socket_lib.dll >nul
copy build\socket_lib.h socket_lib.h >nul
echo Build successful!
echo Generated: socket_lib.dll and socket_lib.h
