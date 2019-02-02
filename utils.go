package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type errExt struct {
	Explain string
	Err     error
}

func (e *errExt) Error() string {
	return e.Err.Error()
}

// LogRespWriter - just wrapper for errors from http.ResponseWriter.Write()
func LogRespWriter(n int, err error) {
	if err != nil {
		Logz("Write failed: %v, %v", n, err)
	}
}

// wWriter - just wrapper for w.Write([]byte)
func wWrite(w io.Writer, data []byte) {
	LogRespWriter(w.Write(data))
	Logz("Response: %s", data)
}

// Logz - wrapper for debug logs
func Logz(format string, a ...interface{}) {
	fmt.Printf(format+"\n", a...)
}

// LogErr - wrapper for panic logs
func LogErr(err errExt) {
	Logz(err.Explain)
	panic(err)
}

func withouterrJSONMarshal(i interface{}) []byte {
	sliceBytes, err := json.Marshal(i)
	if err != nil {
		//LogErr(errExt{"can't marshal array", err})
		Logz(err.Error())
	}
	return sliceBytes
}

func withouterrIOClose(c io.Closer) {
	err := c.Close()
	if err != nil {
		//LogErr(errExt{"can't close IO", err})
		Logz("can't close IO: %s", err.Error())
	}
}

func osGetENV(key, value string) string {

	if envValue := os.Getenv(key); envValue != "" {
		return envValue
	}
	return value

}
