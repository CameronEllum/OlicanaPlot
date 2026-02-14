@echo off
REM Build CSV IPC Plugin
go build -ldflags="-w -s -H windowsgui" -o csv_reader.exe .
