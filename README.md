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
        "vorona/docker_swarm_deploy_webhook:latest": "webhook-latest"
      },
      "APISecretKey": "WebhookSecretKeyChangeME"
    }

Create Base64 encoded string:

    $ CONFIG=`cat  /tmp/config.json | base64 -w0`
    $ echo $CONFIG
    ewoJICAiUHJpdmF0ZVJlZ2lzdHJ5IjogewoJCSAgICAgICJ1c2VybmFtZSI6ICJ1c2VyIiwKCQkgICAgICAicGFzc3dvcmQiOiAic3VwZXJzZWNyZXRwYXNzd29yZCIsCgkJICAgICAgInNlcnZlcmFkZHJlc3MiOiAibXktZG9ja2VyLXJlZ2lzdHJ5LnByaXZhdGUtaG9zdC5jb20iCgkJICAgIH0sCgkJICAiU2VydmljZXMiOiB7CgkJCSAgICAgICJteS1kb2NrZXItcmVnaXN0cnkucHJpdmF0ZS1ob3N0LmNvbS9wcm9qZWN0cS1hcHA6bGF0ZXN0IjogInByb2plY3RxLXN0YWNrLWxhdGVzdF9iYWNrZW5kIiwKCQkJICAgICAgIm15LWRvY2tlci1yZWdpc3RyeS5wcml2YXRlLWhvc3QuY29tL3Byb2plY3RxLWFwcDpzdGFnZSI6ICJwcm9qZWN0cS1zdGFjay1zdGFnZV9iYWNrZW5kIiwKCQkJICAgICAgInZvcm9uYS9kb2NrZXJfc3dhcm1fZGVwbG95X3dlYmhvb2s6bGF0ZXN0IjogIndlYmhvb2stbGF0ZXN0IgoJCQkgICAgfSwKCQkgICJBUElTZWNyZXRLZXkiOiAiV2ViaG9va1NlY3JldEtleUNoYW5nZU1FIgp9Cgo=
    

Deploy your docker-deploy-webhook to a swarm:

    docker service create --name webhook-latest --constraint "node.role==manager" --publish=8081:8081 \
       --mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
       -e DDW_CONFIG="$CONFIG" \
       vorona/docker_swarm_deploy_webhook:latest


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

    docker build -t vorona/docker_swarm_deploy_webhook .
    
    # push to docker hub:
    docker login # paste credentialt to hub.docker.com
    docker push    vorona/docker_swarm_deploy_webhook:latest
    
    # push to docker registry:
    docker login my-docker-registry.private-host.com # paste credentialt to you private registry
    docker push  my-docker-registry.private-host.com/docker_swarm_deploy_webhook:latest

## Manually update:
    docker service update --image vorona/docker_swarm_deploy_webhook:latest webhook-latest
    docker service update --env-add DDW_CONFIG="$CONFIG" webhook-latest
