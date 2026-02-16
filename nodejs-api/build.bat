@echo off
cd ..
go build -buildmode=c-shared -o nodejs-api/broker_lib.dll ./nodejs-api/broker_lib.go
cd nodejs-api
