ECR_ENDPOINT := 749049578452.dkr.ecr.us-west-2.amazonaws.com
ECR_REPO := $(ECR_ENDPOINT)/eks-demo-app

.PHONY: push
push: build
	aws ecr get-login-password | docker login -u AWS --password-stdin $(ECR_ENDPOINT)
	docker push $(ECR_REPO)

.PHONY: build
build:
	docker build -t $(ECR_REPO) .
