package main

import (
	"docker.io/go-docker/api/types"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// go test -coverprofile=/tmp/cover.out && go tool cover -html=/tmp/cover.out
// socat  -v  UNIX-LISTEN:/tmp/123.sock UNIX:/var/run/docker.sock

const (
	configENVName   = "DDW_CONFIG"
	defaultHTTPAddr = ":8081"

	// APIWebHookKeyName - key-name for secret key
	APIWebHookKeyName = "key"
	// APIEndpointWebHookRegistry - just endpoint
	APIEndpointWebHookRegistry = "/webhook/registry/"
	// APIEndpointWebHookDockerHub - just endpoint
	APIEndpointWebHookDockerHub = "/webhook/dockerhub/"
)

// CR is Case Response structure
type CR map[string]interface{}

type mainConfig struct {
	PrivateRegistry types.AuthConfig
	Services        map[string]string // map[fullImageName]swarmServiceName
	APISecretKey    string
}

func main() {

	if _, err := startService(osGetENV("S_HOST", defaultHTTPAddr)); err.Err != nil {
		LogErr(err)
	}
}

func startService(addr string) (*http.Server, errExt) {
	Logz("Staring webhookd service. %s", time.Now())
	rawConfig, err := base64.URLEncoding.DecodeString(os.Getenv(configENVName))
	if err != nil {
		return nil, errExt{fmt.Sprintf("can't decode base64 value ENV[%s]: %s", configENVName, os.Getenv(configENVName)), err}
	}
	Logz("got environ param %s", configENVName)
	var config mainConfig
	err = json.Unmarshal(rawConfig, &config)
	if err != nil {
		return nil, errExt{fmt.Sprintf("can't decode json value: '%s'", rawConfig), err}
	}
	swarmUpdateOpts := types.ServiceUpdateOptions{
		QueryRegistry:    true,
		RegistryAuthFrom: createBase64AuthData(config.PrivateRegistry),
	}
	Logz("unmarshaled environ param %s", configENVName)
	mux := http.NewServeMux()
	s := &http.Server{Addr: addr, Handler: mux}
	mux.Handle(shutdownEnpoint, &shutdownHandler{s})
	mux.Handle("/", &SwarmServiceHandler{config, swarmUpdateOpts})
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return nil, errExt{fmt.Sprintf("can't bind service to %s", addr), err}
	}
	//Logz("stopping server at %s\n", addr)
	return s, errExt{"OK", nil}
}
