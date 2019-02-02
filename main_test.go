package main

import (
	"bytes"
	"context"
	"docker.io/go-docker/api/types"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	apiEndpointRoot      = "/"
	apiEndpointFake      = "/webhook/fake"
	dockerHostKey        = "DOCKER_HOST"
	payloadDockerService = `{
  "events": [
    {
      "id": "42e24968-662e-4689-ae5e-6a53cd08b5bc",
      "timestamp": "2018-07-12T18:31:17.994407023-04:00",
      "action": "push",
      "target": {
        "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
        "size": 1995,
        "digest": "sha256:791de1ee1a11daaf65379856704197e3a7f64e54cb1a8e8e875b8d658b4adbd2",
        "length": 1995,
        "repository": "projectq-app",
        "url": "https://docker-registry.private-host.com/v2/projectq-app/manifests/sha256:791de1ee1a11daaf65379856704197e3a7f64e54cb1a8e8e875b8d658b4adbd2",
        "tag": "latest"
      },
      "request": {
        "id": "d4b32888-7785-442f-9a58-dd1da5e07477",
        "addr": "80.211.78.147",
        "host": "docker-registry.private-host.com",
        "method": "GET",
        "useragent": "docker/18.03.1-ce go/go1.9.5 git-commit/9ee9f40 kernel/3.10.0-862.3.3.el7.x86_64 os/linux arch/amd64"
      },
      "actor": {
        "name": "vorona"
      },
      "source": {
        "addr": "docker-registry.private-host.com:5000",
        "instanceID": "a0935527-e906-40a1-918a-7e9f0ff5e32b"
      }
    }
  ]
}`
	payloadDockerHub = `{
  "callback_url": "https://registry.hub.docker.com/u/svendowideit/testhook/hook/2141b5bi5i5b02bec211i4eeih0242eg11000a/",
  "push_data": {
    "images": [
        "27d47432a69bca5f2700e4dff7de0388ed65f9d3fb1ec645e2bc24c223dc1cc3",
        "51a9c7c1f8bb2fa19bcd09789a34e63f35abb80044bc10196e304f6634cc582c",
        "..."
    ],
    "pushed_at": 1.417566161e+09,
    "pusher": "trustedbuilder",
    "tag": "latest"
  },
  "repository": {
    "comment_count": "0",
    "date_created": 1.417494799e+09,
    "description": "",
    "dockerfile": "#\n# BUILD\u0009\u0009docker build -t svendowideit/apt-cacher .\n# RUN\u0009\u0009docker run -d -p 3142:3142 -name apt-cacher-run apt-cacher\n#\n# and then you can run containers with:\n# \u0009\u0009docker run -t -i -rm -e http_proxy http://192.168.1.2:3142/ debian bash\n#\nFROM\u0009\u0009ubuntu\n\n\nVOLUME\u0009\u0009[\/var/cache/apt-cacher-ng]\nRUN\u0009\u0009apt-get update ; apt-get install -yq apt-cacher-ng\n\nEXPOSE \u0009\u00093142\nCMD\u0009\u0009chmod 777 /var/cache/apt-cacher-ng ; /etc/init.d/apt-cacher-ng start ; tail -f /var/log/apt-cacher-ng/*\n",
    "full_description": "Docker Hub based automated build from a GitHub repo",
    "is_official": false,
    "is_private": true,
    "is_trusted": true,
    "name": "testhook",
    "namespace": "svendowideit",
    "owner": "svendowideit",
    "repo_name": "svendowideit/testhook",
    "repo_url": "https://registry.hub.docker.com/u/svendowideit/testhook/",
    "star_count": 0,
    "status": "Active"
  }
}`
)

type Case struct {
	Method  string
	Path    string
	Query   string
	Payload string
	DHost   string
	DResp   []DResp
	Status  int
	Result  interface{}
}

type DResp struct {
	statusCode   int
	responseBody []byte
}

var (
	client     = &http.Client{Timeout: time.Second}
	testConfig = mainConfig{
		PrivateRegistry: types.AuthConfig{
			Username:      "vorona",
			Password:      "thixie6loh9Uemier8hoh0se",
			ServerAddress: "docker-registry.private-host.com",
		},
		Services: map[string]string{
			"docker-registry.private-host.com/projectq-app:latest": "projectq-stack-latest_backend",
			"vorona/docker-deploy-webhook:latest":                  "docker-deploy-webhook",
		},
		APISecretKey: "EF3rf34g3gfR2G3r3grf",
	}
	testUpdateOpts = types.ServiceUpdateOptions{
		QueryRegistry:    false,
		RegistryAuthFrom: createBase64AuthData(testConfig.PrivateRegistry),
	}
)

func TestServiceHandler_ServeHTTP(t *testing.T) {
	config := testConfig
	ts := httptest.NewServer(&SwarmServiceHandler{config, testUpdateOpts})

	cases := []Case{
		{ // case 0 check non-POST method
			Method: http.MethodGet,
			Status: http.StatusMethodNotAllowed,
			Result: CR{
				"error": "bad method",
			},
		},
		{ // case 1 check POST without endpoint
			Method: http.MethodPost,
			Status: http.StatusBadRequest,
			Result: CR{
				"error": "bad endpoint",
			},
		},
		{ // case 2 check POST with not allowed endpoint #1
			Path:   apiEndpointFake,
			Method: http.MethodPost,
			Status: http.StatusBadRequest,
			Result: CR{
				"error": "bad endpoint",
			},
		},
		{ // case 3 check POST with not allowed endpoint #2
			Path:   apiEndpointRoot,
			Method: http.MethodPost,
			Status: http.StatusBadRequest,
			Result: CR{
				"error": "bad endpoint",
			},
		},
		{ // case 4 check POST with webhook endpoint without key #1
			Path:   APIEndpointWebHookRegistry,
			Method: http.MethodPost,
			Status: http.StatusForbidden,
			Result: CR{
				"error": "unauthorized",
			},
		},
		{ // case 5 check POST with webhook endpoint without key #2
			Path:   APIEndpointWebHookRegistry,
			Method: http.MethodPost,
			Query:  fmt.Sprintf("%s=%s", APIWebHookKeyName, "fake"),
			Status: http.StatusForbidden,
			Result: CR{
				"error": "unauthorized",
			},
		},
		{ // case 6 check POST with webhook endpoint with right key. Empty payload
			Path:   APIEndpointWebHookRegistry,
			Method: http.MethodPost,
			Query:  fmt.Sprintf("%s=%s", APIWebHookKeyName, config.APISecretKey),
			Status: http.StatusBadRequest,
			Result: CR{
				"error": "can't decode payload: EOF",
			},
		},
	}

	runTests(t, ts, cases, config)
}

func TestPayloadParsingRegistry(t *testing.T) {
	config := testConfig
	ts := httptest.NewServer(&SwarmServiceHandler{config, testUpdateOpts})

	cases := []Case{
		{ // case 0
			Path:    APIEndpointWebHookRegistry,
			Method:  http.MethodPost,
			Query:   fmt.Sprintf("%s=%s", APIWebHookKeyName, config.APISecretKey),
			Status:  http.StatusBadRequest,
			Payload: "{}",
			Result: CR{
				"error": "can't decode payload: payload without events",
			},
		},
		{ // case 1
			Path:    APIEndpointWebHookRegistry,
			Method:  http.MethodPost,
			Query:   fmt.Sprintf("%s=%s", APIWebHookKeyName, config.APISecretKey),
			Status:  http.StatusBadRequest,
			Payload: "{Bad json}",
			Result: CR{
				"error": "can't decode payload: invalid character 'B' looking for beginning of object key string",
			},
		},
		{ // case 2
			Path:    APIEndpointWebHookRegistry,
			Method:  http.MethodPost,
			Query:   fmt.Sprintf("%s=%s", APIWebHookKeyName, config.APISecretKey),
			Status:  http.StatusBadRequest,
			Payload: payloadDockerService,
			DHost:   "unix:///var/run/fake.sock",
			Result: CR{
				"error": "can't connect to service projectq-stack-latest_backend: Cannot connect to the Docker daemon at unix:///var/run/fake.sock. Is the docker daemon running?",
			},
		},
		{ // case 3
			Path:    APIEndpointWebHookRegistry,
			Method:  http.MethodPost,
			Query:   fmt.Sprintf("%s=%s", APIWebHookKeyName, config.APISecretKey),
			Status:  http.StatusBadRequest,
			Payload: payloadDockerService,
			DHost:   "http://:65666",
			Result: CR{
				"error": "can't connect to service projectq-stack-latest_backend: error during connect: Get http://:65666/v1.33/services/projectq-stack-latest_backend?insertDefaults=false: dial tcp: address 65666: invalid port",
			},
		},
	}

	runTests(t, ts, cases, config)
}

func TestPayloadParsingRegistry2ndConfig(t *testing.T) {
	config := mainConfig{}
	ts := httptest.NewServer(&SwarmServiceHandler{config, testUpdateOpts})

	cases := []Case{
		{ // TestPayloadParsingRegistry2ndConfig case 0
			Path:    APIEndpointWebHookRegistry,
			Method:  http.MethodPost,
			Query:   fmt.Sprintf("%s=%s", APIWebHookKeyName, config.APISecretKey),
			Status:  http.StatusOK,
			Payload: payloadDockerService,
			Result: CR{
				"error": "empty ServiceName, exit. IMG: docker-registry.private-host.com/projectq-app:latest",
			},
		},
		{ // TestPayloadParsingRegistry2ndConfig case 1
			Path:    APIEndpointWebHookRegistry,
			Method:  http.MethodPost,
			Query:   fmt.Sprintf("%s=%s", APIWebHookKeyName, config.APISecretKey),
			Status:  http.StatusBadRequest,
			Payload: `{"events": [{"action": "pull"}]}`,
			Result: CR{
				"error": "can't decode payload: PULL is an excluded method",
			},
		},
	}

	runTests(t, ts, cases, config)
}

func TestPayloadParsingHub(t *testing.T) {
	config := testConfig
	ts := httptest.NewServer(&SwarmServiceHandler{config, testUpdateOpts})

	cases := []Case{
		{ // case 0
			Path:    APIEndpointWebHookDockerHub,
			Method:  http.MethodPost,
			Query:   fmt.Sprintf("%s=%s", APIWebHookKeyName, config.APISecretKey),
			Status:  http.StatusOK,
			Payload: payloadDockerHub,
			Result: CR{
				"error": "empty ServiceName, exit. IMG: svendowideit/testhook:latest",
			},
		},
		{ // case 1
			Path:    APIEndpointWebHookDockerHub,
			Method:  http.MethodPost,
			Query:   fmt.Sprintf("%s=%s", APIWebHookKeyName, config.APISecretKey),
			Status:  http.StatusBadRequest,
			Payload: "{Bad json}",
			Result: CR{
				"error": "can't decode payload: invalid character 'B' looking for beginning of object key string",
			},
		},
	}

	runTests(t, ts, cases, config)
}

func TestBadConfigs(t *testing.T) {

	cases := map[string]string{
		"e30K1": "illegal base64 data at input byte 4",
		"e3":    "illegal base64 data at input byte 0",
		"":      "unexpected end of JSON input",
	}
	for idx, item := range cases {

		runChechconfigENVName(t, idx, item)

	}

}

func TestShutdownHandler_ServeHTTP(t *testing.T) {
	if err := os.Setenv(configENVName, convertInterfaceToBase64String(testConfig)); err != nil {
		t.Errorf("can't set OS environ with error: %s", err)
	}
	url := "127.0.0.1:9876"
	go func() {
		_, errS := startService(url)
		if errS.Err != nil {
			t.Errorf("start service error: %v", errS)
			return
		}
	}()
	time.Sleep(20 * time.Millisecond)
	req, errR := http.NewRequest("GET", "http://"+url+shutdownEnpoint, nil)
	if errR != nil {
		t.Errorf("request001 error: %v", errR)
		return
	}
	_, err := client.Do(req)
	if err != nil {
		t.Errorf("responce001 error: %v", err)
		return
	}
	//Logz("%v", resp.Status)
	_, err = client.Do(req)
	if err != nil {
		return
	}
	t.Errorf("responce002 error: %v", err)
}

func TestErrorsStartService(t *testing.T) {
	url := "127.0.0.1:9876"
	l, errL := net.Listen("tcp4", url)
	if errL != nil {
		t.Errorf("tcp4 %s already binded: %v", url, errL)
		return
	}
	_, errS01 := startService(url)
	if errS01.Err == nil {
		t.Errorf("service started w/o error, expected error")
		return
	}
	l.Close()
}

func runTests(t *testing.T, ts *httptest.Server, cases []Case, cfg mainConfig) {

	for idx, item := range cases {
		runCase(t, ts, idx, item, cfg)
	}
}

func runCase(t *testing.T, ts *httptest.Server, idx int, item Case, cfg mainConfig) {
	os.Remove(dockerSimpleSocket)
	var (
		err    error
		result interface{}
		req    *http.Request
	)

	listenerSimpleSocketServer, e := startSimpleSocketServer(dockerSimpleSocket, item.DResp)
	defer withouterrIOClose(listenerSimpleSocketServer)
	if e != nil {
		LogErr(errExt{"can't start startSimpleSocketServer:", e})
	}
	caseName := fmt.Sprintf("case %d: [%s] %s %s", idx, item.Method, item.Path, item.Query)
	url := ts.URL + item.Path + "?" + item.Query

	if err01 := os.Setenv(configENVName, convertInterfaceToBase64String(cfg)); err01 != nil {
		t.Errorf("can't set OS environ with error: %s", err01)
	}
	if err02 := os.Setenv(dockerHostKey, item.DHost); err02 != nil {
		t.Errorf("can't set OS environ with error: %s", err02)
	}

	if item.Method == http.MethodPost {
		reqBody := strings.NewReader(item.Payload)
		req, _ = http.NewRequest(item.Method, url, reqBody)
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, _ = http.NewRequest(item.Method, url, nil)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("[%s] request error: %v", caseName, err)
		return
	}
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != item.Status {
		t.Errorf("[%s] expected http status %v, got %v", caseName, item.Status, resp.StatusCode)
		t.Errorf("[%s] got %s", caseName, body)
		return
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Errorf("[%s] cant unpack json: %v", caseName, err)
		return
	}
	body, _ = json.Marshal(result)
	expected, _ := json.Marshal(item.Result)

	if !bytes.Equal(body, expected) {
		t.Errorf("[%d] results not match\nGot     : %#v\nExpected: %#v", idx, string(body), string(expected))
		return
	}

}

func runChechconfigENVName(t *testing.T, testBase64String, expectedError string) {
	url := "127.0.0.1:9876"
	if err := os.Setenv(configENVName, testBase64String); err != nil {
		t.Errorf("can't set OS environ with error: %s", err)
	}

	if s, errS := startService(url); errS.Err == nil {
		t.Errorf("service started w/o error, expected error")
		s.Shutdown(context.Background())
	} else {
		if expectedError != errS.Error() {
			t.Errorf("expected error: %s \n got error: %s", expectedError, errS.Error())
		}
	}
}

func convertInterfaceToBase64String(i interface{}) string {
	data, err := json.Marshal(i)
	if err != nil {
		LogErr(errExt{"can't marshal interface", err})
	}
	return base64.URLEncoding.EncodeToString(data)
}

func TestM(t *testing.T) {
	if err := os.Setenv("S_HOST", "-1"); err != nil {
		t.Errorf("can't set OS environ with error: %s", err)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected recovery from panic error")
		}
	}()
	main()
}
