VERSION :=	$(shell cat ../VERSION)
NG :=		./node_modules/.bin/ng

all: ./src/environments/version.ts
	$(NG) build --prod --aot

install-deps:
	npm install

serve: ./src/environments/version.ts
	$(NG) serve --port 4300 --proxy-config proxy-dev.conf.json

./src/environments/version.ts: ../VERSION ./src/environments/version.ts.in
	sed -e "s#@version@#$(VERSION)#g" ./src/environments/version.ts.in > \
		./src/environments/version.ts

clean:
	rm -rf dist

distclean:
	rm -rf dist node_modules
