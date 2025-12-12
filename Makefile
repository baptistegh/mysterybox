default: install generate build

install: 
	go mod tidy

fmt:
	go fmt -w -l -s

generate:
	go generate ./...

build:
	go build -o bin/mysterybox cmd/main.go

run: install 
	go run ./cmd/main.go