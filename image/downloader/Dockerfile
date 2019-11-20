FROM golang:1.13-buster as builder
WORKDIR /go/src/code.cloudfoundry.org/eirini-staging
COPY . .
RUN GO111MODULE=on GOOS=linux go build -a -mod vendor -o /downloader cmd/downloader/downloader.go

ARG GIT_SHA
RUN if [ -z "$GIT_SHA" ]; then echo "GIT_SHA not set"; exit 1; else : ; fi

FROM cloudfoundry/cflinuxfs3

WORKDIR /workspace

RUN mkdir -p /packs

USER vcap

COPY --from=builder /downloader /packs/

ENTRYPOINT [ \
  "/packs/downloader" \
]

ARG GIT_SHA
LABEL org.opencontainers.image.revision=$GIT_SHA \
      org.opencontainers.image.source=https://code.cloudfoundry.org/eirini-staging
