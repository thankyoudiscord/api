OUTPUT := birthday-backend
MAIN := cmd/thankyoudiscord/main.go

$(OUTPUT): deps $(wildcard */*/*.go)
	go build -o $(OUTPUT) $(MAIN)

build: $(OUTPUT)

.PHONY: clean
clean:
	go clean
	rm -f $(OUTPUT)

.PHONY: deps
deps: go.mod
	go get ./...

.PHONY: dev
dev:
	@go run *.go
