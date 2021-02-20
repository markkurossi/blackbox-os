GO := go
ALL_TARGETS := wasm/kernel.wasm httpd/httpd wasm/fs	\
wasm/bin/echo.wasm wasm/bin/sh.wasm wasm/bin/ssh.wasm
PUBLIC := mrossi@isle-of-wight.dreamhost.com:markkurossi.com/blackbox-os/

all: $(ALL_TARGETS)

.PHONY: $(ALL_TARGETS)

clean:
	$(RM) $(ALL_TARGETS)

wasm/kernel.wasm: kernel/kernel.go
	cd kernel; GOOS=js GOARCH=wasm $(GO) build -o ../wasm/$(notdir $@)

wasm/bin/sh.wasm: bin/sh/main.go
	cd $(dir $+); GOOS=js GOARCH=wasm $(GO) build -o ../../$@

wasm/bin/echo.wasm: bin/echo/main.go
	cd $(dir $+); GOOS=js GOARCH=wasm $(GO) build -o ../../$@

wasm/bin/ssh.wasm: bin/ssh/main.go
	cd $(dir $+); GOOS=js GOARCH=wasm $(GO) build -o ../../$@

httpd/httpd: httpd/httpd.go
	cd httpd; $(GO) build -o $(notdir $@)

wasm/fs:
	rsync -av sample/fs/.backup/* wasm/fs

rsync:
	rsync -avhe ssh --delete --exclude-from rsync.exclude wasm/* $(PUBLIC)
