build:
	CGO_ENABLED=0 go build  -ldflags="-X 'main.Version=$$(git describe --tags --always --dirty)' -s -w" -o cobweb .
docker: build
	docker build . -t shynome/cobweb:$$(git describe --tags --always --dirty)
push: docker
	docker push shynome/cobweb:$$(git describe --tags --always --dirty)
