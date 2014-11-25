package jsonrequest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	netUrl "net/url"
	"time"
)

const (
	// TODO use compilation flags to active debug mode
	// http://dave.cheney.net/2014/09/28/using-build-to-switch-between-debug-and-release
	debug = false
)

type Request interface {
	Do(method string, endpoint string, requestBody interface{}, jsonResponse interface{}) (StatusCode, error)
}

type RequestClient struct {
	httpClient *http.Client
	baseUrl    string
}

type StatusCode int

func NewRequest(baseUrl string) Request {
	return NewRequestWithTimeout(baseUrl, 10*time.Second)
}

func NewRequestWithTimeout(baseUrl string, timeout time.Duration) Request {
	dialTimeout := func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, timeout)
	}

	transport := http.Transport{
		Dial: dialTimeout,
	}
	client := http.Client{
		Transport: &transport,
	}

	return &RequestClient{baseUrl: baseUrl, httpClient: &client}
}

func (r *RequestClient) Do(method string, endpoint string, requestBody interface{}, jsonResponse interface{}) (StatusCode, error) {
	var err error
	var response *http.Response

	url, err := netUrl.Parse(r.baseUrl + endpoint)
	if err != nil {
		err = fmt.Errorf("JSONrequest: [%s|%s]: Can not build url given the parameters: %s\n", method, r.baseUrl+endpoint, err)
		return 0, err
	}

	requestBytes := make([]byte, 0)
	if requestBody != nil {
		requestBytes, err = json.Marshal(&requestBody)
		if err != nil {
			err = fmt.Errorf("JSONrequest: [%s|%s]: Can not encode request body: %s\n", method, url.String(), err)
			return 0, err
		}

	}
	clientReq, err := http.NewRequest(method, url.String(), bytes.NewReader(requestBytes))
	if err != nil {
		err = fmt.Errorf("JSONrequest: [%s|%s]: Can not create http client: %s\n", method, url.String(), err)
		return 0, err
	}

	clientReq.Header.Add("Content-Type", "application/json")
	clientReq.Header.Add("Accept", "application/json")
	response, err = r.httpClient.Do(clientReq)
	status := ""
	if response != nil {
		defer response.Body.Close()
		status = response.Status
	}
	if err != nil {
		err = fmt.Errorf("JSONrequest: [%s|%s]- [%s]: %s\n", method, url.String(), status, err)
		return 0, err
	}

	// ReadAll allow me to easy debug the output. But for performance is better with -> json.NewDecoder(response.Body)
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("JSONrequest: [%s|%s]- [%s]: Error reading the body Json: %s\n, Request BODY %v\n Response BODY --\n %v", method, url.String(), status, err, string(requestBytes), string(body))
		return StatusCode(response.StatusCode), err
	}

	if debug {
		log.Printf("DEBUG: JSONrequest: [%s|%s] - [%s] \nRequest +++++++\n%v\n Response +++++++\n%v\n", method, url.String(), status, string(requestBytes), string(body))
	}

	if len(body) > 0 {
		err = json.Unmarshal(body, &jsonResponse)
		if err != nil {
			err = fmt.Errorf("JSONrequest: [%s|%s]- [%s]: Error marshalling Json response: %s\n, Request BODY %v\n Response BODY --\n %v", method, url.String(), status, err, string(requestBytes), string(body))
			return StatusCode(response.StatusCode), err
		}
	}

	//decoder := json.NewDecoder(response.Body)
	//err = decoder.Decode(&jsonResponse)

	// server errors
	if response.StatusCode >= 500 && response.StatusCode <= 599 {
		err = fmt.Errorf("JSONrequest: [%s|%s] - [%s]\n", method, url, status)
	}
	return StatusCode(response.StatusCode), err
}
