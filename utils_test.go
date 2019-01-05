package main

import (
	"github.com/pkg/errors"
	"os"
	"testing"
)

func TestLogz(t *testing.T) {
	err := errExt{"explain test err", errors.New("test err")}
	Logz("Test %s\n", "OK")
	LogRespWriter(0, err.Err)
	h := SwarmServiceHandler{}
	_, err.Err = h.getHookParamsFromPayload(nil, "/fake/")
	if err.Err == nil {
		t.Errorf("bad error")
	}
}

func TestFuncWithoutErrJsonMarshal(t *testing.T) {
	// Just for coverage testing
	// catching error on json.Marshal()

	_ = withouterrJSONMarshal(make(chan int))
}

func TestFuncWithoutErrIOClose(t *testing.T) {
	// Just for coverage testing
	// catching error on f.Close()

	f, _ := os.Create("/file/path/name.txt")
	withouterrIOClose(f)
}

func TestFuncOsGetENV(t *testing.T) {
	// Just for coverage testing
	// catching error on f.Close()

	if val := osGetENV("NOTrealKeY", defaultHTTPAddr); val != defaultHTTPAddr {
		t.Errorf("something wrong\nGOT: %s\nEXP: %s", val, defaultHTTPAddr)
	}
}
