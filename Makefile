tag = latest

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=linux go build -installsuffix cgo -o webhook .
	docker build -t moolen/187:$(tag) .

.PHONY: push
push: build
	docker push moolen/187:$(tag)
