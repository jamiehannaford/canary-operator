GOFILES:=$(shell find . -name '*.go' | grep -v -E '(./vendor)')

all: \
	build \
	release \
	redeploy

build: $(GOFILES)
	GOOS=linux GOARCH=amd64 go build github.com/jamiehannaford/canary-operator/cmd/operator

release:
	@docker build -t jamiehannaford/canary-operator -f build/Dockerfile .
	@docker push jamiehannaford/canary-operator

redeploy:
	@kubectl delete -n kube-system deployment canary-operator
	@kubectl create -f build/deployment.yaml

.PHONY: build release redeploy
