#go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
lint:
	golangci-lint run ./...

test:
	go test -v ./e2e/...

up:
	docker-compose up -d

down:
	docker-compose down

rebuild:
	docker-compose up -d --build
