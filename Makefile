VERSION :=	$(shell cat VERSION)

GIT_REV	:=	$(shell git rev-parse --short HEAD)
GIT_BRANCH :=	$(shell git symbolic-ref --short HEAD)

.PHONY:		dist

all:
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@
	cp go/maker .

install-deps:
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@

clean:
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@
	rm -rf dist
	find . -name \*~ -delete

distclean: clean
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@

#
# Release building.
#

GOOS     := $(shell go env GOOS)
GOARCH   := $(shell go env GOARCH)

DISTOS	 := $(GOOS)
ifeq ($(GOOS), darwin)
DISTOS   := macos
endif

DISTARCH := $(GOARCH)
ifeq ($(GOARCH), amd64)
DISTARCH := x86_64
endif

dist: DISTNAME := maker-$(VERSION)-$(DISTOS)-$(DISTARCH)
dist: DIR      := ../dist/$(DISTNAME)
dist:
	rm -rf dist/$(DIR) && mkdir -p dist/$(DIR)
ifndef SKIP_WEBAPP
	cd webapp && $(MAKE)
endif
	GOARCH=$(GOARCH) DIR=../dist/$(DIR) $(MAKE) -C go
	cp README.md LICENSE.txt ./dist/$(DIR)
ifeq ($(GIT_BRANCH), master)
	echo $(GIT_REV) > ./dist/$(DIR)/.commit_id
endif
	(cd dist && zip -r $(DISTNAME).zip $(DISTNAME))

