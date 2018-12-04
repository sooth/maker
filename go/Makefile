VERSION :=	$(shell cat ../VERSION)

GOPATH ?=	$(HOME)/go
CGO_ENABLED :=	1
GO_TAGS :=	json1

APP :=		maker

.PHONY:		$(APP)

all: $(APP)

$(APP): EXE := $(shell go env GOEXE)
$(APP): BIN ?= $(APP)$(EXE)
$(APP): DIR ?= .
$(APP):
	test -e ../webapp/dist && $(GOPATH)/bin/packr -v -z || true
	CGO_ENABLED=1 go build -o $(DIR)/$(BIN) --tags "$(GO_TAGS)"

install-deps:
	go get -u github.com/gobuffalo/packr/...
	go mod download

clean:
	rm -f $(APP)
	find . -name \*~ -delete
	find . -name \*-packr.go -delete

distclean: clean
