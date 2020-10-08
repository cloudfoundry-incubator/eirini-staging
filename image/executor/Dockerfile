ARG baseimage=eirini/eirinifs:test
FROM golang:1.13-buster as builder
WORKDIR /go/src/code.cloudfoundry.org/eirini-staging
COPY . .
RUN GO111MODULE=on GOOS=linux go build -mod vendor -a -o /executor cmd/executor/executor.go

ARG GIT_SHA
RUN if [ -z "$GIT_SHA" ]; then echo "GIT_SHA not set"; exit 1; else : ; fi

FROM ${baseimage}

RUN mkdir -p /home/vcap/workspace /home/vcap/packs /home/vcap/cache

WORKDIR /home/vcap/workspace

COPY --from=builder /executor /home/vcap/packs/

ENTRYPOINT [ \
  "/home/vcap/packs/executor" \
]

ARG GIT_SHA
LABEL org.opencontainers.image.revision=$GIT_SHA \
      org.opencontainers.image.source=https://code.cloudfoundry.org/eirini-staging
