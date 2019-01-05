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
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/app_cgo .

pub: build_cgo
	docker build -t ddw .
	docker tag ddw docker-registry.private-host.com/docker_deploy_webhook:${TAG}
	docker push docker-registry.private-host.com/docker_deploy_webhook:${TAG}

publish: build_cgo
	docker build -t ddw .
	docker tag ddw vorona/docker_deploy_webhook:${TAG}
	docker push vorona/docker_deploy_webhook:${TAG}
