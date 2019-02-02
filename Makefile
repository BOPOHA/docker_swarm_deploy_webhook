TAG="latest"

all: clean test lint

clean:
	rm build/* -rf

lint:
	gometalinter

test:
	go test -v -cover

cover:
	mkdir -p build
	go test -coverprofile=build/cover.out && go tool cover -html=build/cover.out

gometalinter:
	mkdir -p ~/bin
	go get github.com/alecthomas/gometalinter
	go build -o ~/bin/gometalinter github.com/alecthomas/gometalinter
	gometalinter --install

build:
	go build -o build/app_cgo .

build_cgo:
	docker build -t vorona/docker_swarm_deploy_webhook .

pub: build_cgo
	docker tag vorona/docker_swarm_deploy_webhook docker-registry.private-host.com/docker_deploy_webhook:${TAG}
	docker push                                   docker-registry.private-host.com/docker_deploy_webhook:${TAG}

publish: build_cgo
	docker push vorona/docker_swarm_deploy_webhook:${TAG}
