export CFLAGS := -O3 -Wall -std=gnu11
export CC := x86_64-w64-mingw32-gcc
WINDRES := x86_64-w64-mingw32-windres
export CGO_ENABLED := 1
export GOOS := windows
export GOARCH := amd64

DEPLOYMENT_HOST ?= winvm
DEPLOYMENT_PATH ?= Desktop

all: wireguard.exe

deps/.prepared:
	mkdir -p deps/gotmpdir deps/gocache deps/gopath
	ln -sf "$$(go env GOROOT)" deps/goroot
	touch $@

resources.syso: resources.rc manifest.xml ui/icon/icon.ico
	$(WINDRES) -i $< -o $@ -O coff

rwildcard=$(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2) $(filter $(subst *,%,$2),$d))
wireguard.exe: resources.syso $(call rwildcard,,*.go *.c *.h) deps/.prepared
	GOROOT=$(PWD)/deps/goroot \
	GOCACHE=$(PWD)/deps/gocache \
	GOPATH=$(PWD)/deps/gopath \
	GOTMPDIR=$(PWD)/deps/gotmpdir \
	go build -ldflags="-H windowsgui -s -w" -gcflags=all=-trimpath=$(PWD) -asmflags=all=-trimpath=$(PWD) -v -o $@

deploy: wireguard.exe
	-ssh $(DEPLOYMENT_HOST) -- 'taskkill /im wireguard.exe /f'
	scp wireguard.exe $(DEPLOYMENT_HOST):$(DEPLOYMENT_PATH)

clean:
	chmod -R u+w deps/gopath/pkg/mod
	rm -rf resources.syso wireguard.exe deps

.PHONY: deploy clean all
