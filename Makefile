MAKEFLAGS=--no-builtin-rules --no-builtin-variables --always-make

fmt:
	gofumports -local github.com/perchcredit/gqlgenc -w .

lint:
	golangci-lint run

test:
	go test -v ./...
