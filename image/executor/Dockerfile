FROM golang:1.12.2-stretch as builder
WORKDIR /go/src/code.cloudfoundry.org/eirini-staging
COPY . .
RUN GOOS=linux go build -a -o /executor cmd/executor/executor.go

FROM cloudfoundry/cflinuxfs3

WORKDIR /workspace

RUN mkdir -p /packs

COPY --from=builder /executor /packs/

ENTRYPOINT [ \
  "/packs/executor" \
]
