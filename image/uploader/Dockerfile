FROM golang:1.13-stretch as builder
WORKDIR /go/src/code.cloudfoundry.org/eirini-staging
COPY . .
RUN GO111MODULE=on GOOS=linux go build -mod vendor -a -o /uploader cmd/uploader/uploader.go

FROM cloudfoundry/cflinuxfs3

WORKDIR /workspace

RUN mkdir -p /packs

COPY --from=builder /uploader /packs/

ENTRYPOINT [ \
  "/packs/uploader" \
]
