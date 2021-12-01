.PHONY: build
build:
	go build ./...

.PHONY: vet
vet:
	go vet ./... 

.PHONY: golint
golint:
	golint ./...

.PHONY: staticcheck
staticcheck:
	staticcheck ./...

# Run tests
test:  vet
	go test ./  -covermode=atomic -coverprofile=coverage.txt