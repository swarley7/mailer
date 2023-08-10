#!/bin/bash
env GOOS=windows GOARCH=amd64 go build -o builds/mailer_win_amd64.exe mailer.go
env GOOS=darwin GOARCH=amd64 go build -o builds/mailer_macOS_amd64 mailer.go
env GOOS=linux GOARCH=amd64 go build -o builds/mailer_linux_amd64 mailer.go
