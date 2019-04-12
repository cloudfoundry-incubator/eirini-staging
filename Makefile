IMAGE_USER ?= eirini
IMAGE_TAG ?= latest

test:
	./scripts/test.sh

integration-test:
	./scripts/integration-test.sh

image:
	./scripts/image.sh $(IMAGE_USER) $(IMAGE_TAG)

deploy:
	./scripts/deploy.sh $(DEPLOY_USER) $(DEPLOY_TAG)

.PHONY: test integration-test image deploy
