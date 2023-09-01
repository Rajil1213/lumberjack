.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test -cover ./...	

.PHONY: cover
cover:
	go test -v -coverprofile coverage/cover.out ./... &&\
	go tool cover -html coverage/cover.out -o coverage/cover.html &&\
	open coverage/cover.html
