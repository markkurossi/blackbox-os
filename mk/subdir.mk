GO1.11 := $(HOME)/work/go/bin/go1.11rc2
phony_targets = build vet

.PHONY: $(phony_targets)

build:
	GOOS=js GOARCH=wasm $(GO1.11) build

vet:
	GOOS=js GOARCH=wasm $(GO1.11) vet
