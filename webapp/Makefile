NG :=	./node_modules/.bin/ng

all:
	$(NG) build --prod --aot

install-deps:
	npm install

serve:
	$(NG) serve --port 4300 --proxy-config proxy-dev.conf.json
