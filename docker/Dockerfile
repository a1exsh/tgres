FROM registry.opensource.zalan.do/stups/ubuntu:16.04-60

EXPOSE 2003 2004 8125 8888

# gettext-base provides envsubst(1)

RUN apt-get update && \
    apt-get -y install gettext-base && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

ENV GOPATH "/usr/local"

RUN mkdir -m 777 /var/log/tgres

RUN mkdir $GOPATH/{bin,etc}

COPY tgres.conf.template $GOPATH/etc/
COPY run-tgres.sh $GOPATH/bin/

COPY tgres $GOPATH/bin

CMD $GOPATH/bin/run-tgres.sh
