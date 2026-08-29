@echo off

SET GOPROXY=https://goproxy.cn

SET CGO_ENABLED=0
SET GOOS=windows
SET GOARCH=amd64

go build -ldflags "-extldflags '-static'" -o excel2pb.exe main.go
