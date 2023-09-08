APP=dnstap
IMAGE=dnstap
TAG?=latest
DOCKER_ROOT?=andrewstuart
NAMESPACE=default

FQTAG=$(DOCKER_ROOT)/$(IMAGE):$(TAG)

SHA=$(shell docker inspect --format "{{ index .RepoDigests 0 }}" $(1))

test:
	go test ./...

go:
	GOOS=linux CGO_ENABLED=0 go build -o app

docker: go test
	docker build -t $(FQTAG) . 
	docker push $(FQTAG)

deploy: docker
	kubectl apply --namespace $(NAMESPACE) -f k8s.yaml
	kubectl --namespace $(NAMESPACE) set image deployment/$(APP) $(APP)=$(call SHA,$(FQTAG))
