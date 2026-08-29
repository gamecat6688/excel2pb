@echo off

SET GOPROXY=https://goproxy.cn

SET CGO_ENABLED=0
SET GOOS=linux
SET GOARCH=amd64
go build -o excel2pb

SET CGO_ENABLED=0
SET GOOS=windows
SET GOARCH=amd64
go build -o excel2pb.exe

SET CGO_ENABLED=0
SET GOOS=darwin
SET GOARCH=amd64
go build -o excel2pb_mac

pause
