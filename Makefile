lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

test:
	go test ./... -coverprofile=coverage.out

security-check:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec ./...

complexity-check:
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	gocyclo -over 10 .
