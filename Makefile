# make && ./bin/biliroaming-go-server 直接执行编译后的二进制文件

# init project path
HOMEDIR := $(shell pwd)
OUTDIR  := $(HOMEDIR)/output

APPNAME := $(shell basename `pwd`)
OUTPUT_FILE := ${APPNAME}.tar.gz

# init command params
GO      := $(shell which go)
GOPATH  := $(shell $(GO) env GOPATH)
GOMOD   := $(GO) mod
GOBUILD := $(GO) build
GOTEST  := $(GO) test -gcflags="-N -l"
GOPKGS  := $$($(GO) list ./...| grep -vE "vendor")

# make, make all
all: prepare compile package

# make prepare, download dependencies
prepare: gomod
gomod: set-env 

# set proxy env
set-env:
	$(GO) env -w GOPROXY="https://goproxy.io,direct"

# make compile
compile: build
build:
	$(GOBUILD) -o $(HOMEDIR)/bin/$(APPNAME)
	$(shell cd $(HOMEDIR) && rm -f $(SCRIPT_TARGET) && cd $(HOMEDIR))

# make test, test your code
test: prepare test-case
test-case:
	$(GOTEST) -v -cover $(GOPKGS)

# package stage: make package
package: package-bin
package-bin:
	$(shell rm -rf $(OUTDIR))
	$(shell mkdir -p $(OUTDIR))
	$(shell cp -a bin $(OUTDIR)/bin)
	$(shell cp -a database $(OUTDIR)/database)
	$(shell cp -a entity $(OUTDIR)/entity)
	$(shell cp -a models $(OUTDIR)/models)
	$(shell cp -a sql $(OUTDIR)/sql)
	$(shell cp -a config.yml $(OUTDIR)/config.yml)
	$(shell cd $(OUTDIR)/; tar -zcf ${OUTPUT_FILE} ./*; rm -rf bin)

# clean stage: make clean
clean:
	rm -rf $(OUTDIR)

# avoid filename conflict and speed up build 
.PHONY: all prepare compile test package clean build