VERSION :=	$(shell cat ../VERSION)
GIT_REV	:=	$(shell git rev-parse --short HEAD)
BRANCH :=	$(shell git symbolic-ref --short HEAD)

GOPATH ?=	$(HOME)/go
CGO_ENABLED :=	1
GO_TAGS :=	json1

PACKAGE :=	gitlab.com/crankykernel/maker/go
LDFLAGS :=	-X \"$(PACKAGE)/version.GitRevision=$(GIT_REV)\" \
		-X \"$(PACKAGE)/version.Version=$(VERSION)\" \
		-X \"$(PACKAGE)/version.GitBranch=$(BRANCH)\"

APP :=		maker

.PHONY:		$(APP)

all: $(APP)

$(APP): EXE := $(shell go env GOEXE)
$(APP): BIN ?= $(APP)$(EXE)
$(APP): DIR ?= .
$(APP):
	test -e ../webapp/dist && GO111MODULE=on $(GOPATH)/bin/packr2 -v
	CGO_ENABLED=1 go build -o $(DIR)/$(BIN) \
		-ldflags "$(LDFLAGS)" \
		-tags "$(GO_TAGS)"

install-deps:
	go get github.com/gobuffalo/packr/v2/packr2

gofmt:
	go fmt ./...

clean:
	$(GOPATH)/bin/packr2 clean
	rm -f $(APP)
	find . -name \*~ -delete
	find . -name \*-packr.go -delete

distclean: clean
