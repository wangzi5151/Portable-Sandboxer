@echo off
setlocal
if not exist dist mkdir dist
set GOOS=windows
set GOARCH=amd64
set GOARCH=arm64
set GOOS=linux
set GOARCH=amd64
set GOARCH=arm64
echo Cross builds complete. Native Fyne GUI packaging requires platform toolchains.
