@echo off

SET GOPROXY=https://goproxy.cn

SET CGO_ENABLED=0
SET GOHOSTARCH=linux
SET GOARCH=amd64
go build -o excel2pb

SET CGO_ENABLED=0
SET GOHOSTARCH=windows 
SET GOARCH=amd64
go build -o excel2pb.exe

SET CGO_ENABLED=0
SET GOHOSTARCH=darwin 
SET GOARCH=amd64
go build -o excel2pb_mac

pause