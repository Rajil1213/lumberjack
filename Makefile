GOBIN ?= $$(go env GOPATH)/bin

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test -cover ./...	

.PHONY: install-go-test-coverage
install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

.PHONY: check-coverage
check-coverage: install-go-test-coverage
	go test ./... -coverprofile=./coverage/cover.out -covermode=atomic -coverpkg=./...
	${GOBIN}/go-test-coverage --config=./.testcoverage.yml

.PHONY: cover
cover:
	go test -v -coverprofile coverage/cover.out ./... &&\
	go tool cover -html coverage/cover.out -o coverage/cover.html &&\
	open coverage/cover.html
