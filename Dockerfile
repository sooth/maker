# First stage build.

# Build using a Nodejs provided base image, and add Go and other
# dependencies ourselves.

FROM node:10.13-slim

RUN apt update && \
    apt -y install \
    	make \
	zip \
	git \
	gcc \
	libsqlite3-dev

RUN cd /usr/local && \
    curl -L https://dl.google.com/go/go1.11.2.linux-amd64.tar.gz | tar zxf -
ENV PATH=/usr/local/go/bin:$PATH

COPY / /maker
WORKDIR /maker
RUN make install-deps
RUN make dist
RUN make

# Stage 2.
FROM node:10.13-slim
COPY --from=0 /maker/maker /usr/local/bin/maker
VOLUME /data
WORKDIR /data
CMD ["/usr/local/bin/maker", "server"]
