APP=dnstap
IMAGE=dnstap
TAG?=latest
DOCKER_ROOT?=docker.io/andrewstuart
NAMESPACE=dnstap

FQTAG=$(DOCKER_ROOT)/$(IMAGE):$(TAG)

# SHA=$(shell buildah inspect --format "{{ index .RepoDigests 0 }}" $(1))

test:
	go test ./...

go:
	GOOS=linux CGO_ENABLED=0 go build -o app

docker: go test
	buildah bud -t $(FQTAG) . 
	buildah push --digestfile sha.txt $(FQTAG)
	# Push again with SHA because buildah is weird
	# buildah push $(call SHA,$(FQTAG))

deploy: docker
	kubectl apply --namespace $(NAMESPACE) -f k8s.yaml
	kubectl --namespace $(NAMESPACE) set image deployment/$(APP) $(APP)=$(DOCKER_ROOT)/$(IMAGE)@$(shell cat sha.txt)
