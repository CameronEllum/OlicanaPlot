@echo off
REM Build Template IPC Plugin
go build -ldflags="-w -s -H windowsgui" -o template-go.exe .
