FROM golang:1.12-alpine
LABEL maintainer "Rafael Martins <rafael@rafaelmartins.eng.br>"

ADD . /code

RUN set -x \
    && apk add --no-cache --virtual .build-deps \
        git \
    && ( \
        cd /code \
        && go build -o /usr/bin/yatr \
    ) \
    && rm -rf /code \
    && apk del .build-deps

ENTRYPOINT ["/usr/bin/yatr"]
