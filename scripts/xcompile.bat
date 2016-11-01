@echo off

if exist ..\bin rd /s /q ..\bin
mkdir ..\bin\win64
mkdir ..\bin\linux64

SET GOOS=windows
SET GOARCH=amd64
go build -o ..\bin\win64\go-alpr.exe ../main.go
echo "Built application for Windows/amd64"

SET GOOS=linux
SET GOARCH=amd64
go build -o ..\bin\linux64\go-alpr ../main.go
echo "Built application for Linux/amd64"
