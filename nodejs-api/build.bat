@echo off
go build -buildmode=c-shared -o broker_lib.dll broker_lib.go
