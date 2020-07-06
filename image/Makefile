IMAGES = downloader executor uploader

TAG ?= latest
IMAGE_DIR := ${CURDIR}
EIRINI_STAGING_DIR := ${IMAGE_DIR}/..
REVISION := $(shell git -C $(EIRINI_STAGING_DIR) rev-parse HEAD)

.PHONY: $(IMAGES)

all: $(IMAGES)

$(IMAGES):
	docker build $(EIRINI_STAGING_DIR) \
		--file "$(IMAGE_DIR)/$@/Dockerfile" \
		--build-arg GIT_SHA=$(REVISION) \
		--tag "eirini/recipe-$@:$(TAG)"

push:
	$(foreach img, $(IMAGES), docker push eirini/recipe-$(img):$(TAG);)

push-%:
	docker push eirini/recipe-$*:$(TAG)

clean:
	docker image prune --force
	docker rmi $(shell docker image ls --format="{{.Repository}}:{{.Tag}}" --filter=reference="eirini/*")

