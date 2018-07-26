GO1.11 := $(HOME)/work/go/bin/go1.11beta2
ophony_targets = build

.PHONY: $(phony_targets)

build:
	GOOS=js GOARCH=wasm $(GO1.11) build
