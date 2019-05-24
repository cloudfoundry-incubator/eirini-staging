FROM golang:1.12.2-stretch as builder
WORKDIR /go/src/code.cloudfoundry.org/eirini-staging
COPY . .
RUN GOOS=linux go build -a -o /downloader cmd/downloader/downloader.go

FROM eirinistaging/cf

WORKDIR /workspace

RUN mkdir -p /packs

COPY --from=builder /downloader /packs/

ENTRYPOINT [ \
  "/packs/downloader" \
]
