VERSION :=	$(shell cat VERSION)

.PHONY:		dist

all:
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@

install-deps:
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@

dist: GOOS   := $(shell go env GOOS)
dist: GOARCH := $(shell go env GOARCH)
dist: DIR    := ../dist/maker-$(VERSION)-$(GOOS)-$(GOARCH)
dist:
	rm -rf dist/$(OUTDIR) && mkdir -p dist/$(OUTDIR)
	cd webapp && $(MAKE)
	GOARCH=$(GOARCH) DIR=../dist/$(DIR) $(MAKE) -C go
	cp README.md LICENSE.txt ./dist/$(DIR)
	(cd dist && zip -r $(DIR).zip $(DIR))

clean:
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@
	rm -rf dist

distclean: clean
	cd webapp && $(MAKE) $@
	cd go && $(MAKE) $@
