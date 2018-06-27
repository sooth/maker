VERSION :=	$(shell cat VERSION)

CGO_ENABLED :=	1

TAGS :=		json1

.PHONY:		dist

all:
	cd webapp && $(MAKE) $@
	packr -z
	CGO_ENABLED=1 go build --tags "$(TAGS)"

install-deps:
	cd webapp && $(MAKE) $@
	dep ensure

dist: GOOS=$(shell go env GOOS)
dist: GOARCH=$(shell go env GOARCH)
dist: GOEXE=$(shell go env GOEXE)
dist: OUTDIR=maker-$(VERSION)-$(GOOS)-$(GOARCH)
dist: OUTBIN=maker$(GOEXE)
dist:
	dep ensure
	rm -rf dist/$(OUTDIR)
	mkdir -p dist/$(OUTDIR)
	cd webapp && $(MAKE)
	packr -z
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build --tags "$(TAGS)" -o dist/$(OUTDIR)/$(OUTBIN)
	(cd dist && zip -r $(OUTDIR).zip $(OUTDIR))

clean:
	rm -f maker maker.exe
	rm -rf dist
	cd webapp && $(MAKE) $@

distclean: clean
	rm -rf vendor
	cd webapp && $(MAKE) $@

dev-server:
	reflex -d none -s -r \.go$$ -- go run --tags "json1" ./main.go server
