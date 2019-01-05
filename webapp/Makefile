VERSION :=	$(shell cat ../VERSION)
GIT_REV	:=	$(shell git rev-parse --short HEAD)
NG :=		./node_modules/.bin/ng

all: update-version
	$(NG) build --prod --aot

install-deps:
	npm install

update-version:
	sed -e "s#@VERSION@#$(VERSION)#g" \
	    -e "s#@GIT_REVISION@#$(GIT_REV)#g" \
		./src/environments/version.ts.in > \
		./src/environments/version.ts

clean:
	rm -rf dist

distclean:
	rm -rf dist node_modules
