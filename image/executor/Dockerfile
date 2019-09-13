ARG baseimage=cloudfoundry/cflinuxfs3

FROM golang:1.12-stretch as builder
WORKDIR /go/src/code.cloudfoundry.org/eirini-staging
COPY . .
RUN GO111MODULE=on GOOS=linux go build -mod vendor -a -o /executor cmd/executor/executor.go

FROM ${baseimage}

WORKDIR /workspace

RUN mkdir -p /packs

COPY --from=builder /executor /packs/

ENTRYPOINT [ \
  "/packs/executor" \
]
