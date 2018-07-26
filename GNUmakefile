GO1.11 := $(HOME)/work/go/bin/go1.11beta2
ALL_TARGETS := wasm/kernel.wasm httpd/httpd

all: $(ALL_TARGETS)

.PHONY: $(ALL_TARGETS)

clean:
	$(RM) $(ALL_TARGETS)

wasm/kernel.wasm: kernel/kernel.go
	cd kernel; GOOS=js GOARCH=wasm $(GO1.11) build -o ../wasm/$(notdir $@)

httpd/httpd: httpd/httpd.go
	cd httpd; $(GO1.11) build -o $(notdir $@)
