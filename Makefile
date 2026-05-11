.PHONY: test
test:
	go clean -testcache && go test ./...

.PHONY: cover
cover:
	go test -cover ./...

.PHONY: coverhtml
coverhtml:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
