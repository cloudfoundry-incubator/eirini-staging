ARG baseimage=cloudfoundry/cflinuxfs3

FROM golang:1.13-buster as builder
WORKDIR /go/src/code.cloudfoundry.org/eirini-staging
COPY . .
RUN GO111MODULE=on GOOS=linux go build -mod vendor -a -o /uploader cmd/uploader/uploader.go

ARG GIT_SHA
RUN if [ -z "$GIT_SHA" ]; then echo "GIT_SHA not set"; exit 1; else : ; fi

FROM ${baseimage}

ARG user_id=2000
ARG group_id=2000

RUN mkdir -p /packs /workspace
RUN chown ${user_id}:${group_id} /packs /workspace

WORKDIR /workspace

USER ${user_id}:${group_id}

COPY --from=builder /uploader /packs/

ENTRYPOINT [ \
  "/packs/uploader" \
]

ARG GIT_SHA
LABEL org.opencontainers.image.revision=$GIT_SHA \
      org.opencontainers.image.source=https://code.cloudfoundry.org/eirini-staging
