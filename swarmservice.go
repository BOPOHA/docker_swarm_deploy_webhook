package main

import (
	"context"
	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"docker.io/go-docker/api/types/swarm"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/distribution/notifications"
	"io"
	"io/ioutil"
	"net/http"
)

var allowedWebHookEndpoints = map[string]bool{
	APIEndpointWebHookRegistry:  true,
	APIEndpointWebHookDockerHub: true,
}

// DockerHubPayload - payload from docker registry webhook service
type DockerHubPayload struct {
	Repository struct {
		RepoName string `json:"repo_name"`
	}
	PushData struct {
		Tag string
	} `json:"push_data"`
}

// DockerRegistryV2Payload - payload from docker registry webhook service
type DockerRegistryV2Payload map[string][]notifications.Event

func (h *SwarmServiceHandler) updateService(params HookParamsFromPayload) error {
	// Logz("Starting update service with values: %+v\n", params)
	// TODO: need separate func for validate params
	if len(params.serviceName)*len(params.registryImage) == 0 {
		return fmt.Errorf("nothing to do, exit. SN: %s IMG: %s", params.serviceName, params.registryImage)
	}
	ctx := context.Background()

	authBytes := withouterrJSONMarshal(h.config.PrivateRegistry)
	authBase64 := base64.URLEncoding.EncodeToString(authBytes)

	if cli, err := docker.NewEnvClient(); err == nil {
		defer withouterrIOClose(cli)
		if service, _, errCliService := cli.ServiceInspectWithRaw(
			ctx, params.serviceName, types.ServiceInspectOptions{}); errCliService == nil {
			spec := &service.Spec
			spec.TaskTemplate.ContainerSpec.Image = params.registryImage
			if respServiceUpdate, errCliServiceUpd := cli.ServiceUpdate(
				ctx,
				service.ID,
				swarm.Version{Index: service.Version.Index},
				*spec,
				types.ServiceUpdateOptions{
					//QueryRegistry:       true,
					EncodedRegistryAuth: authBase64,
					//RegistryAuthFrom: "spec",
				}); errCliServiceUpd == nil {
				fmt.Println(respServiceUpdate.Warnings)
				message := "SERVICE UPDATED - " + params.serviceName + " " + service.ID
				fmt.Println(message)
			} else {
				return fmt.Errorf("updating a service: %s, %s", service.ID, errCliServiceUpd)
			}
		} else {
			return fmt.Errorf("can't connect to service %s: %s", params.serviceName, errCliService)
		}
	} else {
		return fmt.Errorf("can't connect to docker host: %s", err)
	}
	return nil
}

func (h *SwarmServiceHandler) getHookParamsFromPayload(body io.Reader, endpoint string) (HookParamsFromPayload, error) {
	var params HookParamsFromPayload

	if endpoint == APIEndpointWebHookRegistry {
		payload := DockerRegistryV2Payload{}

		rawBody, errRe := ioutil.ReadAll(body)
		if errRe != nil {
			return params, errRe
		}
		err := json.Unmarshal(rawBody, &payload)
		if err != nil {
			return params, err
		}

		//decoder := json.NewDecoder(body)
		//if err := decoder.Decode(&payload); err != nil {
		//	return params, err
		//}
		if events := payload["events"]; len(events) > 0 {
			firstEvent := events[0]
			if firstEvent.Action == "pull" {
				return params, errors.New("PULL is an excluded method")
			}
			Logz("%s", rawBody)
			Logz("%+v", payload)
			params.registryImage = firstEvent.Request.Host + "/" + firstEvent.Target.Repository + ":" + firstEvent.Target.Tag
			params.serviceName = h.config.Services[params.registryImage]
			return params, nil
		}
		return params, errors.New("payload without events")
	}

	if endpoint == APIEndpointWebHookDockerHub {
		payload := DockerHubPayload{}
		decoder := json.NewDecoder(body)
		if err := decoder.Decode(&payload); err != nil {
			return params, err
		}
		Logz("%+v", payload)
		params.registryImage = payload.Repository.RepoName + ":" + payload.PushData.Tag
		params.serviceName = h.config.Services[params.registryImage]
		return params, nil
	}

	return HookParamsFromPayload{}, errors.New("invalid endpoint")

}

// HookParamsFromPayload - golint
type HookParamsFromPayload struct {
	registryImage string
	serviceName   string
}

// SwarmServiceHandler - main http handler
type SwarmServiceHandler struct {
	config mainConfig
}

func (h *SwarmServiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Logz("%s %s %s %s %s %v\n", r.Method, r.RequestURI, r.Proto, r.RemoteAddr, r.Host, r.ContentLength)
	if r.Method == "POST" {
		if allowedWebHookEndpoints[r.URL.Path] {
			values := r.URL.Query()
			keys := values[APIWebHookKeyName]
			if len(keys) > 0 {
				if key := keys[0]; key == h.config.APISecretKey {
					// do your staff here
					plParams, err := h.getHookParamsFromPayload(r.Body, r.URL.Path)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						data := withouterrJSONMarshal(CR{
							"error": "can't decode payload: " + err.Error(),
						})
						LogRespWriter(w.Write(data))
						return
					}
					Logz("%+v", plParams)
					if plParams.serviceName == "" {
						// we have to response with 2xx code here, because of error:
						// retryingsink: error writing events: httpSink{http://callback.url}: response status 400 Bad Request unaccepted, retrying
						w.WriteHeader(http.StatusOK)
						resp := withouterrJSONMarshal(CR{
							"error": fmt.Sprintf("empty ServiceName, exit. IMG: %s", plParams.registryImage),
						})
						LogRespWriter(w.Write(resp))
						return
					}
					// UPDATING SERVICE:
					if err := h.updateService(plParams); err == nil {
						w.WriteHeader(http.StatusOK)
						LogRespWriter(w.Write([]byte(`{"status": "OK"}`)))
					} else {
						w.WriteHeader(http.StatusBadRequest)
						data := withouterrJSONMarshal(CR{
							"error": err.Error(),
						})
						LogRespWriter(w.Write(data))
					}
					return
				}
			}
			http.Error(w, `{"error": "unauthorized"}`, http.StatusForbidden)
		} else {
			http.Error(w, `{"error":"bad endpoint"}`, http.StatusBadRequest)
		}

	} else {
		http.Error(w, `{"error":"bad method"}`, http.StatusMethodNotAllowed)
	}
}
