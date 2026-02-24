.PHONY: test
test:
	CGO_ENABLED=0 go test -v ./...

.PHONY: ui
ui:
	CGO_ENABLED=0 go run cmd/demo/main.go
