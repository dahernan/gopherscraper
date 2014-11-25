package jsonrequest

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGET(t *testing.T) {
	Convey("GET request", t, func() {
		ts := httptest.NewServer(http.HandlerFunc(jsonServer(jsonBuilder)))
		url := ts.URL
		defer ts.Close()
		request := NewRequest(url)
		var jsonResponse map[string]interface{}

		status, err := request.Do("GET", "/123", nil, &jsonResponse)
		So(err, ShouldBeNil)
		So(status, ShouldEqual, 200)
		json := make(map[string]interface{})
		json["message"] = "hello"

		So(jsonResponse["message"], ShouldEqual, json["message"])
	})
}

func TestPOST(t *testing.T) {
	Convey("POST request", t, func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			So(req.Method, ShouldEqual, "POST")
			So(req.URL.String(), ShouldEqual, "/hello/world")
			So(req.Header.Get("Content-Type"), ShouldEqual, "application/json")
			So(req.Header.Get("Accept"), ShouldEqual, "application/json")

			body, err := ioutil.ReadAll(req.Body)
			So(err, ShouldBeNil)
			defer req.Body.Close()
			jsonMap := make(map[string]string)
			err = json.Unmarshal(body, &jsonMap)
			So(err, ShouldBeNil)
			So(jsonMap["one"], ShouldEqual, "1 one")
			So(jsonMap["two"], ShouldEqual, "2 two")
			sendOK(w)
		}))

		url := ts.URL
		defer ts.Close()
		request := NewRequest(url)

		jsonMap := make(map[string]string)
		jsonMap["one"] = "1 one"
		jsonMap["two"] = "2 two"

		var jsonResponse map[string]interface{}

		status, err := request.Do("POST", "/hello/world", jsonMap, &jsonResponse)

		So(err, ShouldBeNil)
		So(status, ShouldEqual, 200)

		So(jsonResponse["ok"], ShouldEqual, true)
	})
}

func Test404(t *testing.T) {
	Convey("404 error", t, func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			sendNotFound(w)
		}))

		url := ts.URL
		defer ts.Close()
		request := NewRequest(url)

		status, err := request.Do("GET", "/hello/world", nil, nil)

		So(err, ShouldBeNil)
		So(status, ShouldEqual, 404)
	})
}

func Test400(t *testing.T) {
	Convey("400 error", t, func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			sendBadRequest(w)
		}))

		url := ts.URL
		defer ts.Close()
		request := NewRequest(url)

		status, err := request.Do("GET", "/hello/world", nil, nil)

		So(err, ShouldBeNil)
		So(status, ShouldEqual, 400)
	})
}

func Test500(t *testing.T) {
	Convey("500 error", t, func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			sendInternalServerError(w)
		}))

		url := ts.URL
		defer ts.Close()
		request := NewRequest(url)

		status, err := request.Do("GET", "/hello/world", nil, nil)

		So(err, ShouldNotBeNil)
		So(status, ShouldEqual, 500)
	})
}

func TestJsonErrorMarshall(t *testing.T) {
	Convey("Error marshalling Json on the server", t, func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			writeJsonBytes(w, []byte("bad json document"))

		}))
		url := ts.URL
		defer ts.Close()
		request := NewRequest(url)
		var jsonResponse map[string]interface{}
		_, err := request.Do("GET", "/", nil, jsonResponse)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "invalid character 'b' looking for beginning of value")
	})
}

func TestUrlError(t *testing.T) {
	Convey("Error in the url", t, func() {
		request := NewRequest("wrong")
		response, err := request.Do("GET", "/", nil, nil)
		So(err, ShouldNotBeNil)
		t.Log(err)
		So(response, ShouldEqual, 0)
	})
}

type httpHandlerFunc func(w http.ResponseWriter, req *http.Request)
type jsonHttpBuilderFunc func(req *http.Request) interface{}

func jsonBuilder(req *http.Request) interface{} {
	jsonMap := make(map[string]interface{})
	name := req.URL.Query().Get(":name")
	jsonMap["message"] = "hello" + name
	return jsonMap
}

func sendOK(w http.ResponseWriter) {
	jsonMap := make(map[string]interface{})
	jsonMap["ok"] = true
	jsonBytes, _ := json.Marshal(jsonMap)
	writeJsonBytes(w, jsonBytes)
}

func sendNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	jsonMap := make(map[string]interface{})
	jsonMap["exists"] = false
	jsonBytes, _ := json.Marshal(jsonMap)
	writeJsonBytes(w, jsonBytes)
}

func sendNotFoundWithNilObject(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	writeJsonBytes(w, nil)
}

func sendBadRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	jsonMap := make(map[string]interface{})
	jsonMap["error"] = "Bad request error"
	jsonBytes, _ := json.Marshal(jsonMap)
	writeJsonBytes(w, jsonBytes)
}

func sendInternalServerError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	jsonMap := make(map[string]interface{})
	jsonMap["error"] = "Internal Server Error"
	jsonBytes, _ := json.Marshal(jsonMap)
	writeJsonBytes(w, jsonBytes)
}

func writeJsonBytes(w http.ResponseWriter, jsonBytes []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(jsonBytes)))
	w.Write(jsonBytes)
}

func jsonServer(builderFunc jsonHttpBuilderFunc) (hanlderFunc httpHandlerFunc) {
	hanlderFunc = func(w http.ResponseWriter, req *http.Request) {
		jsonObject := builderFunc(req)
		jsonBytes, _ := json.Marshal(jsonObject)
		writeJsonBytes(w, jsonBytes)
	}
	return
}
