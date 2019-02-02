package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

const dockerSimpleSocket = "/tmp/echo.sock"

type FakeService struct {
	Programm []DResp
	N        int
}

func (h *FakeService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//Logz("D: %+v\n\n", r)
	if len(h.Programm) > h.N {
		w.WriteHeader(h.Programm[h.N].statusCode)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Api-Version", "1.38")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Write(h.Programm[h.N].responseBody)
	}
	h.N++
}

func TestSwarmErrorsWithSockerServerFakeTLS(t *testing.T) {
	if err := os.Setenv("DOCKER_CERT_PATH", "/tmp/"); err != nil {
		t.Errorf("can't set OS environ with error: %s", err)
	}
	time.Sleep(20 * time.Millisecond)
	config := testConfig
	ts := httptest.NewServer(&SwarmServiceHandler{config, testUpdateOpts})

	cases := []Case{
		{ // case 0
			Path:    APIEndpointWebHookRegistry,
			Method:  http.MethodPost,
			Query:   fmt.Sprintf("%s=%s", APIWebHookKeyName, config.APISecretKey),
			Status:  http.StatusBadRequest,
			Payload: payloadDockerService,
			Result: CR{
				"error": "can't connect to docker host: Could not load X509 key pair: open /tmp/cert.pem: no such file or directory",
			},
		},
	}

	runTests(t, ts, cases, config)

	//os.Remove(dockerSimpleSocket)
	if err := os.Setenv("DOCKER_CERT_PATH", ""); err != nil {
		t.Errorf("can't set OS environ with error: %s", err)
	}
}

func TestSwarmErrorsWithSockerServerFakeCodes(t *testing.T) {
	time.Sleep(20 * time.Millisecond)
	config := testConfig
	ts := httptest.NewServer(&SwarmServiceHandler{config, testUpdateOpts})

	cases := []Case{
		{ // case 0
			Path:    APIEndpointWebHookRegistry,
			Method:  http.MethodPost,
			Query:   fmt.Sprintf("%s=%s", APIWebHookKeyName, config.APISecretKey),
			Status:  http.StatusBadRequest,
			Payload: payloadDockerService,
			DHost:   "unix://" + dockerSimpleSocket,
			DResp: []DResp{
				{
					statusCode:   http.StatusOK,
					responseBody: []byte(`{"ID":"knmtuvl25atbbpmsra8yl6daz","Version":{"Index":92917},"CreatedAt":"2018-09-15T21:53:48.33970499Z","UpdatedAt":"2018-09-22T16:51:58.819201217Z","Spec":{"Name":"projectq-stack-latest_backend","Labels":{"com.docker.stack.image":"docker-registry.private-host.com/projectq-app","com.docker.stack.namespace":"projectq-stack-latest","traefik.backend":"app","traefik.backend.loadbalancer.swarm":"true","traefik.docker.network":"web","traefik.enable":"true","traefik.frontend.passHostHeader":"true","traefik.frontend.rule":"Host:projectq-002.private-host.com","traefik.port":"8000","traefik.protocol":"http"},"TaskTemplate":{"ContainerSpec":{"Image":"docker-registry.private-host.com/projectq-app:latest","Labels":{"com.docker.stack.namespace":"projectq-stack-latest"},"Privileges":{"CredentialSpec":null,"SELinuxContext":null},"Isolation":"default"},"Resources":{},"RestartPolicy":{"Condition":"on-failure","MaxAttempts":0},"Placement":{"Platforms":[{"Architecture":"amd64","OS":"linux"}]},"Networks":[{"Target":"gruw0a4grxj58zjf2dens9ezk","Aliases":["backend"]},{"Target":"jmrezbnml8vk47h0jz4f5nmjz","Aliases":["backend"]}],"ForceUpdate":0,"Runtime":"container"},"Mode":{"Replicated":{"Replicas":2}},"UpdateConfig":{"Parallelism":1,"Delay":10000000000,"FailureAction":"pause","MaxFailureRatio":0,"Order":"stop-first"},"EndpointSpec":{"Mode":"vip"}},"PreviousSpec":{"Name":"projectq-stack-latest_backend","Labels":{"com.docker.stack.image":"docker-registry.private-host.com/projectq-app","com.docker.stack.namespace":"projectq-stack-latest","traefik.backend":"app","traefik.backend.loadbalancer.swarm":"true","traefik.docker.network":"web","traefik.enable":"true","traefik.frontend.passHostHeader":"true","traefik.frontend.rule":"Host:projectq-002.private-host.com","traefik.port":"8000","traefik.protocol":"http"},"TaskTemplate":{"ContainerSpec":{"Image":"docker-registry.private-host.com/projectq-app:latest@sha256:c72e6f0209f1d7feecb526e9a26fc276577538c4b840278bd9734d8fcb5cd180","Labels":{"com.docker.stack.namespace":"projectq-stack-latest"},"Privileges":{"CredentialSpec":null,"SELinuxContext":null},"Isolation":"default"},"Resources":{},"RestartPolicy":{"Condition":"on-failure","MaxAttempts":0},"Placement":{"Platforms":[{"Architecture":"amd64","OS":"linux"}]},"Networks":[{"Target":"gruw0a4grxj58zjf2dens9ezk","Aliases":["backend"]},{"Target":"jmrezbnml8vk47h0jz4f5nmjz","Aliases":["backend"]}],"ForceUpdate":0,"Runtime":"container"},"Mode":{"Replicated":{"Replicas":2}},"UpdateConfig":{"Parallelism":1,"Delay":10000000000,"FailureAction":"pause","MaxFailureRatio":0,"Order":"stop-first"},"EndpointSpec":{"Mode":"vip"}},"Endpoint":{"Spec":{"Mode":"vip"},"VirtualIPs":[{"NetworkID":"gruw0a4grxj58zjf2dens9ezk","Addr":"10.0.2.181/24"},{"NetworkID":"jmrezbnml8vk47h0jz4f5nmjz","Addr":"10.0.1.202/24"}]},"UpdateStatus":{"State":"completed","StartedAt":"2018-09-16T22:12:23.2894165Z","CompletedAt":"2018-09-16T22:13:00.88211065Z","Message":"update completed"}}`),
				},
				{
					statusCode:   http.StatusBadGateway,
					responseBody: []byte(`{}`),
				},
			},
			Result: CR{
				"error": "updating a service: knmtuvl25atbbpmsra8yl6daz, Error response from daemon: {}",
			},
		},
		{ // case 1
			Path:    APIEndpointWebHookRegistry,
			Method:  http.MethodPost,
			Query:   fmt.Sprintf("%s=%s", APIWebHookKeyName, config.APISecretKey),
			Status:  http.StatusOK,
			Payload: payloadDockerService,
			DHost:   "unix://" + dockerSimpleSocket,
			DResp: []DResp{
				{
					statusCode:   http.StatusOK,
					responseBody: []byte(`{"ID":"knmtuvl25atbbpmsra8yl6daz","Version":{"Index":92917},"CreatedAt":"2018-09-15T21:53:48.33970499Z","UpdatedAt":"2018-09-22T16:51:58.819201217Z","Spec":{"Name":"projectq-stack-latest_backend","Labels":{"com.docker.stack.image":"docker-registry.private-host.com/projectq-app","com.docker.stack.namespace":"projectq-stack-latest","traefik.backend":"app","traefik.backend.loadbalancer.swarm":"true","traefik.docker.network":"web","traefik.enable":"true","traefik.frontend.passHostHeader":"true","traefik.frontend.rule":"Host:projectq-002.private-host.com","traefik.port":"8000","traefik.protocol":"http"},"TaskTemplate":{"ContainerSpec":{"Image":"docker-registry.private-host.com/projectq-app:latest","Labels":{"com.docker.stack.namespace":"projectq-stack-latest"},"Privileges":{"CredentialSpec":null,"SELinuxContext":null},"Isolation":"default"},"Resources":{},"RestartPolicy":{"Condition":"on-failure","MaxAttempts":0},"Placement":{"Platforms":[{"Architecture":"amd64","OS":"linux"}]},"Networks":[{"Target":"gruw0a4grxj58zjf2dens9ezk","Aliases":["backend"]},{"Target":"jmrezbnml8vk47h0jz4f5nmjz","Aliases":["backend"]}],"ForceUpdate":0,"Runtime":"container"},"Mode":{"Replicated":{"Replicas":2}},"UpdateConfig":{"Parallelism":1,"Delay":10000000000,"FailureAction":"pause","MaxFailureRatio":0,"Order":"stop-first"},"EndpointSpec":{"Mode":"vip"}},"PreviousSpec":{"Name":"projectq-stack-latest_backend","Labels":{"com.docker.stack.image":"docker-registry.private-host.com/projectq-app","com.docker.stack.namespace":"projectq-stack-latest","traefik.backend":"app","traefik.backend.loadbalancer.swarm":"true","traefik.docker.network":"web","traefik.enable":"true","traefik.frontend.passHostHeader":"true","traefik.frontend.rule":"Host:projectq-002.private-host.com","traefik.port":"8000","traefik.protocol":"http"},"TaskTemplate":{"ContainerSpec":{"Image":"docker-registry.private-host.com/projectq-app:latest@sha256:c72e6f0209f1d7feecb526e9a26fc276577538c4b840278bd9734d8fcb5cd180","Labels":{"com.docker.stack.namespace":"projectq-stack-latest"},"Privileges":{"CredentialSpec":null,"SELinuxContext":null},"Isolation":"default"},"Resources":{},"RestartPolicy":{"Condition":"on-failure","MaxAttempts":0},"Placement":{"Platforms":[{"Architecture":"amd64","OS":"linux"}]},"Networks":[{"Target":"gruw0a4grxj58zjf2dens9ezk","Aliases":["backend"]},{"Target":"jmrezbnml8vk47h0jz4f5nmjz","Aliases":["backend"]}],"ForceUpdate":0,"Runtime":"container"},"Mode":{"Replicated":{"Replicas":2}},"UpdateConfig":{"Parallelism":1,"Delay":10000000000,"FailureAction":"pause","MaxFailureRatio":0,"Order":"stop-first"},"EndpointSpec":{"Mode":"vip"}},"Endpoint":{"Spec":{"Mode":"vip"},"VirtualIPs":[{"NetworkID":"gruw0a4grxj58zjf2dens9ezk","Addr":"10.0.2.181/24"},{"NetworkID":"jmrezbnml8vk47h0jz4f5nmjz","Addr":"10.0.1.202/24"}]},"UpdateStatus":{"State":"completed","StartedAt":"2018-09-16T22:12:23.2894165Z","CompletedAt":"2018-09-16T22:13:00.88211065Z","Message":"update completed"}}`),
				},
				{
					statusCode:   http.StatusOK,
					responseBody: []byte(`{}`),
				},
			},
			Result: CR{
				"status": "OK",
			},
		},
	}

	runTests(t, ts, cases, config)

	//os.Remove(dockerSimpleSocket)
}

type errReader int

const ioutilReaderTestErrorMsg = "ioreader test error"

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New(ioutilReaderTestErrorMsg)
}

func TestSwarmErrorsWithIoutilReadAll(t *testing.T) {
	h := &SwarmServiceHandler{testConfig, testUpdateOpts}
	_, err := h.getHookParamsFromPayload(errReader(0), APIEndpointWebHookRegistry)
	if err.Error() != ioutilReaderTestErrorMsg {
		t.Errorf("expected error: %s,\ngot: %s", ioutilReaderTestErrorMsg, err.Error())

	}
}

func TestSwarmErrorsWithEmptyHookParamsFromPayload(t *testing.T) {
	h := &SwarmServiceHandler{testConfig, testUpdateOpts}
	cases := []HookParamsFromPayload{
		{"", ""},
		{"test", ""},
		{"", "test"},
	}
	for _, v := range cases {
		err := h.updateService(HookParamsFromPayload{v.registryImage, v.serviceName})
		expectedError := fmt.Sprintf("nothing to do, exit. SN: %s IMG: %s", v.serviceName, v.registryImage)
		if err.Error() != expectedError {
			t.Errorf("expected error: %s,\ngot: %s", expectedError, err.Error())
			return
		}
	}

}

func startSimpleSocketServer(path string, program []DResp) (net.Listener, error) {

	server := http.Server{
		Handler: &FakeService{program, 0},
	}

	unixListener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	go server.Serve(unixListener)
	return unixListener, nil

}
