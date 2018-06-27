FROM centos:7

RUN yum -y install epel-release && \
    yum -y install git \
    	   	   make \
		   zip \
		   gcc \
		   mingw64-gcc \
		   mingw32-gcc

RUN curl -o /tmp/go.tar.gz -L https://dl.google.com/go/go1.10.3.linux-amd64.tar.gz && \
    cd /usr/local && \
    tar xf /tmp/go.tar.gz
ENV PATH=/usr/local/go/bin:$PATH

ENV NVM_DIR="$HOME/.nvm"

RUN mkdir -p ${NVM_DIR} && \
    curl --silent -o - https://raw.githubusercontent.com/creationix/nvm/v0.33.11/install.sh | bash && \
    . ${NVM_DIR}/nvm.sh && \
    nvm install --lts && \
    nvm use --lts

RUN mkdir -p /go/src/gitlab.com/crankykernel/maker
COPY / /go/src/gitlab.com/crankykernel/maker

ENV GOPATH /go
ENV PATH=${GOPATH}:${PATH}

RUN go get -u github.com/gobuffalo/packr/... && \
    go get -u github.com/golang/dep/cmd/dep

COPY ./build/docker-entrypoint.sh /
CMD ["./docker-entrypoint.sh"]