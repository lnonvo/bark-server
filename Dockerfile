FROM golang:1-alpine AS builder

COPY . /go/src/github.com/finb/bark-server

WORKDIR /go/src/github.com/finb/bark-server

ARG VERSION=unknown
ARG COMMIT=unknown

RUN set -ex \
    && apk add gcc libc-dev \
    && BUILD_DATE=$(date "+%F %T") \
    && go install -trimpath -ldflags \
            "-X 'main.version=${VERSION}' \
             -X 'main.buildDate=${BUILD_DATE}' \
             -X 'main.commitID=${COMMIT}' \
             -w -s"

FROM alpine

LABEL maintainer="mritd <mritd@linux.com>"

# set up nsswitch.conf for Go's "netgo" implementation
# - https://github.com/golang/go/blob/go1.9.1/src/net/conf.go#L194-L275
# - docker run --rm debian:stretch grep '^hosts:' /etc/nsswitch.conf
RUN echo 'hosts: files dns' > /etc/nsswitch.conf

# default timezon
# override it with `--build-arg TIMEZONE=xxxx`
ARG TIMEZONE=Asia/Shanghai
ARG GEMINI_KEY=""
ARG GEMINI_MODEL="gemini-3-flash"
ARG WEBHOOK_URL=""

ENV TZ=${TIMEZONE}
ENV BARK_SERVER_GEMINI_KEY=${GEMINI_KEY}
ENV BARK_SERVER_GEMINI_MODEL=${GEMINI_MODEL}
ENV BARK_SERVER_WEBHOOK_URL=${WEBHOOK_URL}

RUN set -ex \
    && apk upgrade \
    && apk add --no-cache bash ca-certificates tzdata \
    && ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone

COPY --from=builder /go/bin/bark-server /usr/local/bin/bark-server
COPY --from=builder /go/src/github.com/finb/bark-server/deploy/entrypoint.sh /entrypoint.sh

VOLUME /data

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]

CMD ["bark-server"]
