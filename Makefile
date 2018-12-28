VERSION :=	$(shell cat VERSION)

.PHONY:		dist

all:
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@

install-deps:
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@

dist: GOOS     := $(shell go env GOOS)
dist: GOARCH   := $(shell go env GOARCH)
dist: DISTNAME := maker-$(VERSION)-$(GOOS)-$(GOARCH)
dist: DIR      := ../dist/$(DISTNAME)
dist:
	rm -rf dist/$(DIR) && mkdir -p dist/$(DIR)
	cd webapp && $(MAKE)
	GOARCH=$(GOARCH) DIR=../dist/$(DIR) $(MAKE) -C go
	cp README.md LICENSE.txt ./dist/$(DIR)
	(cd dist && zip -r $(DISTNAME).zip $(DISTNAME))

clean:
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@
	rm -rf dist

distclean: clean
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@
