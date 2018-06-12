VERSION :=	0.0.1
VSUFFIX :=	dev

CGO_ENABLED :=	1

.PHONY:		dist

all:
	cd webapp && $(MAKE) $@
	packr -z
	CGO_ENABLED=1 go build

install-deps:
	cd webapp && $(MAKE) $@
	dep ensure

dist: GOOS=$(shell go env GOOS)
dist: GOARCH=$(shell go env GOARCH)
dist: GOEXE=$(shell go env GOEXE)
dist: OUTDIR=maker-$(VERSION)$(VSUFFIX)-$(GOOS)-$(GOARCH)
dist: OUTBIN=maker$(GOEXE)
dist:
	dep ensure
	rm -rf dist/$(OUTDIR)
	mkdir -p dist/$(OUTDIR)
	cd webapp && $(MAKE)
	packr -z
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build -o dist/$(OUTDIR)/$(OUTBIN)
	(cd dist && zip -r $(OUTDIR).zip $(OUTDIR))

clean:
	rm -f maker maker.exe

dev-server:
	reflex -d none -s -r \.go$$ -- go run ./main.go server
