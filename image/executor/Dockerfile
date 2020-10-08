ARG baseimage=cloudfoundry/cflinuxfs3
FROM golang:1.13-buster as builder
WORKDIR /go/src/code.cloudfoundry.org/eirini-staging
COPY . .
RUN GO111MODULE=on GOOS=linux go build -mod vendor -a -o /executor cmd/executor/executor.go

ARG GIT_SHA
RUN if [ -z "$GIT_SHA" ]; then echo "GIT_SHA not set"; exit 1; else : ; fi

FROM ${baseimage}

ARG user_id=2000
ARG group_id=2000

RUN mkdir -p /packs /workspace /cache
RUN chown ${user_id}:${group_id} /packs /workspace /cache

WORKDIR /workspace

USER ${user_id}:${group_id}

COPY --from=builder /executor /packs/

ENTRYPOINT [ \
  "/packs/executor" \
]

ARG GIT_SHA
LABEL org.opencontainers.image.revision=$GIT_SHA \
      org.opencontainers.image.source=https://code.cloudfoundry.org/eirini-staging
