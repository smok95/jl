
set GOOS=linux
set GOARCH=amd64

go build -o gtlogview -ldflags "-w -s" ./cmd/jl/main.go
