GO1.11 := $(HOME)/work/go/bin/go1.11rc2
ALL_TARGETS := wasm/kernel.wasm httpd/httpd wasm/fs
PUBLIC := mrossi@isle-of-wight.dreamhost.com:markkurossi.com/blackbox-os/

all: $(ALL_TARGETS)

.PHONY: $(ALL_TARGETS)

clean:
	$(RM) $(ALL_TARGETS)

wasm/kernel.wasm: kernel/kernel.go
	cd kernel; GOOS=js GOARCH=wasm $(GO1.11) build -o ../wasm/$(notdir $@)

httpd/httpd: httpd/httpd.go
	cd httpd; $(GO1.11) build -o $(notdir $@)

wasm/fs:
	rsync -av sample/fs/.backup/* wasm/fs

rsync:
	rsync -avhe ssh --delete --exclude-from rsync.exclude wasm/* $(PUBLIC)
