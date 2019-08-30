GO := go
phony_targets = build vet

.PHONY: $(phony_targets)

build:
	GOOS=js GOARCH=wasm $(GO) build

vet:
	GOOS=js GOARCH=wasm $(GO) vet
