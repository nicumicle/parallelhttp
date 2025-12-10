.PHONY=vet
vet:
	go vet -c=10 ./...

.PHONY=tidy
tidy:
	go mod tidy

.PHONY=test
test:
	go test -v ./...

.PHONY=test-bench
test-bench:
	go test -bench ./...

.PHONY=lint
lint:
	go tool golangci-lint run

.PHONY=build
build:
	go build -o out/parallelhttp cmd/cli/main.go

.PHONY=build-http-server
build-http-server:
	go build -o out/service cmd/service/main.go
