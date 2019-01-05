# Docker Deploy Webhook

A web service for automated deployment of releases to a Docker Swarm, triggered by a webhook from Docker Hub or Docker Registry.
The idea was taken from https://github.com/iaincollins/docker-deploy-webhook

Advantages:
 - written on Golang, 100% code coverage
 - smallest docker image size: 8.44 MB vs 152MB
 - works directly with a docker socket (do not need syscall to /usr/bin/docker)

## Configuration

Supported environment variables:

    S_HOST=":8081" // Port to run on
    DDW_CONFIG="ewogICJQcml2YXRlUmVnaXN0..." // Base64 encoded string with configuration

Create temporary file `/tmp/config.json` with configuration by example:

    {
      "PrivateRegistry": {
        "username": "user",
        "password": "supersecretpassword",
        "serveraddress": "my-docker-registry.private-host.com"
      },
      "Services": {
        "my-docker-registry.private-host.com/projectq-app:latest": "projectq-stack-latest_backend",
        "my-docker-registry.private-host.com/projectq-app:stage": "projectq-stack-stage_backend",
        "vorona/docker-deploy-webhook:latest": "docker-deploy-webhook"
      },
      "APISecretKey": "WebhookSecretKeyChangeME"
    }

Create Base64 encoded string:

    $ cat  /tmp/config.json | base64 -w0
    ewogICJQcml2YXRlUmVnaXN0cnkiOiB7CiAgICAidXNlcm5hbWUiOiAidXNlciIsCiAgICAicGFzc3dvcmQiOiAic3VwZXJzZWNyZXRwYXNzd29yZCIsCiAgICAic2VydmVyYWRkcmVzcyI6ICJteS1kb2NrZXItcmVnaXN0cnkucHJpdmF0ZS1ob3N0LmNvbSIKICB9LAogICJTZXJ2aWNlcyI6IHsKICAgICJteS1kb2NrZXItcmVnaXN0cnkucHJpdmF0ZS1ob3N0LmNvbS9wcm9qZWN0cS1hcHA6bGF0ZXN0IjogInByb2plY3RxLXN0YWNrLWxhdGVzdF9iYWNrZW5kIiwKICAgICJteS1kb2NrZXItcmVnaXN0cnkucHJpdmF0ZS1ob3N0LmNvbS9wcm9qZWN0cS1hcHA6c3RhZ2UiOiAicHJvamVjdHEtc3RhY2stc3RhZ2VfYmFja2VuZCIsCiAgICAidm9yb25hL2RvY2tlci1kZXBsb3ktd2ViaG9vazpsYXRlc3QiOiAiZG9ja2VyLWRlcGxveS13ZWJob29rIgogIH0sCiAgIkFQSVNlY3JldEtleSI6ICJX

Deploy your docker-deploy-webhook to a swarm:

    docker service create --name docker-deploy-webhook --constraint "node.role==manager" --publish=8081:8081 \
       --mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
       -e DDW_CONFIG="ewogICJQcml2YXRlUmVnaXN0cnkiOiB7CiAgICAidXNlcm5hbWUiOiAidXNlciIsCiAgICAicGFzc3dvcmQiOiAic3VwZXJzZWNyZXRwYXNzd29yZCIsCiAgICAic2VydmVyYWRkcmVzcyI6ICJteS1kb2NrZXItcmVnaXN0cnkucHJpdmF0ZS1ob3N0LmNvbSIKICB9LAogICJTZXJ2aWNlcyI6IHsKICAgICJteS1kb2NrZXItcmVnaXN0cnkucHJpdmF0ZS1ob3N0LmNvbS9wcm9qZWN0cS1hcHA6bGF0ZXN0IjogInByb2plY3RxLXN0YWNrLWxhdGVzdF9iYWNrZW5kIiwKICAgICJteS1kb2NrZXItcmVnaXN0cnkucHJpdmF0ZS1ob3N0LmNvbS9wcm9qZWN0cS1hcHA6c3RhZ2UiOiAicHJvamVjdHEtc3RhY2stc3RhZ2VfYmFja2VuZCIsCiAgICAidm9yb25hL2RvY2tlci1kZXBsb3ktd2ViaG9vazpsYXRlc3QiOiAiZG9ja2VyLWRlcGxveS13ZWJob29rIgogIH0sCiAgIkFQSVNlY3JldEtleSI6ICJX" \
       vorona/docker_deploy_webhook:latest


## Configure Docker Hub to use Webhook

Add a webhook to the Docker Hub image repository.

The URL to specify for the webhook in Docker Hub will be `${your-server}/webhook/dockerhub/?key=${your-token}`.

e.g. http://projectq-swarm.private-host.com:8082/webhook/dockerhub/?key=WebhookSecretKeyChangeME

You can configure multiple webhooks for a Docker Hub repository (e.g. one webhook on your production cluster, one on development, etc).

While all webhooks will receive the callback, the specific image that has just been built (e.g. `:latest`, `:edge`, etc.) will only be deployed to an environment if the webhook service running on it has it whitelisted in the `config.json` block for that environment.

## Configure Docker Registry to use Webhook

For example config `/etc/docker-distribution/registry/config.yml` with a webhook:

    version: 0.1
    log:
      fields:
        service: registry:2
    storage:
        cache:
            layerinfo: inmemory
        filesystem:
            rootdirectory: /data/registry
    http:
        addr: 127.0.0.1:5000
    notifications:
      events:
        includereferences: true
      endpoints:
        - name: projectq-001
          disabled: false
          url:   http://projectq-swarm.private-host.com:8082/webhook/registry/?key=WebhookSecretKeyChangeME
          timeout: 1s
          threshold: 10
          backoff: 1s
          ignoredmediatypes:
            - application/octet-stream
          ignore:
            mediatypes:
              - application/octet-stream
            actions:
              - pull

The URL to specify for the webhook in Docker Registry will be `${your-server}/webhook/registry/?key=${your-token}`.

e.g. http://projectq-swarm.private-host.com:8082/webhook/registry/?key=WebhookSecretKeyChangeME

## Testing

To test locally with the example payload:

    curl -v -H "Content-Type: application/json" --data @payload.json  http://localhost:3000/webhook/registry/?key=WebhookSecretKeyChangeME

   
# BUILD

    # go build -o app .
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .
    docker build -t ddw .
    
    # push to docker hub:
    docker login # paste credentialt to hub.docker.com
    docker tag ddw vorona/docker_deploy_webhook:latest
    docker push    vorona/docker_deploy_webhook:latest
    
    # push to docker registry:
    docker login my-docker-registry.private-host.com # paste credentialt to you private registry
    docker tag ddw my-docker-registry.private-host.com/docker_deploy_webhook:latest
    docker push    my-docker-registry.private-host.com/docker_deploy_webhook:latest

## Manually update:
    docker service update --image vorona/docker_deploy_webhook:latest ddw
