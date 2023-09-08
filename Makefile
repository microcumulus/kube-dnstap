APP=dnstap
IMAGE=dnstap
TAG?=latest
DOCKER_ROOT?=docker.io/andrewstuart
NAMESPACE=dnstap

FQTAG=$(DOCKER_ROOT)/$(IMAGE):$(TAG)

SHA=$(shell podman inspect --format "{{ index .RepoDigests 0 }}" $(1))

test:
	go test ./...

go:
	GOOS=linux CGO_ENABLED=0 go build -o app

docker: go test
	podman build -t $(FQTAG) . 
	podman push $(FQTAG)
	# push again but strip localhost/ from the sha name
	podman push $(call SHA,$(FQTAG))

deploy: docker
	kubectl apply --namespace $(NAMESPACE) -f k8s.yaml
	kubectl --namespace $(NAMESPACE) set image deployment/$(APP) $(APP)=$(call SHA,$(FQTAG))
