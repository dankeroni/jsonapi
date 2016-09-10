package jsonapi

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Error struct
type Error struct {
	Error   string `json:"error"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// JSONAPI struct
type JSONAPI struct {
	BaseURL string
	Headers map[string]string
}

// SuccessCallback runs on a successfull request and parse
type SuccessCallback func()

// HTTPErrorCallback runs on a errored HTTP request
type HTTPErrorCallback func(statusCode int, statusMessage, errorMessage string)

// InternalErrorCallback runs on an internal error
type InternalErrorCallback func(error)

var client = &http.Client{}

func (jsonAPI *JSONAPI) request(verb, url string, parameters url.Values,
	requestBody interface{}, responseBody interface{}, onSuccess SuccessCallback,
	onHTTPError HTTPErrorCallback, onInternalError InternalErrorCallback) {
	url = jsonAPI.BaseURL + url + "?" + parameters.Encode()
	var request *http.Request
	var err error
	if requestBody != nil {
		serializedRequestBody, err := json.Marshal(requestBody)
		if err != nil {
			onInternalError(err)
			return
		}

		serializedRequestBodyReader := bytes.NewReader(serializedRequestBody)
		request, err = http.NewRequest(verb, url, serializedRequestBodyReader)
	} else {
		request, err = http.NewRequest(verb, url, nil)
	}
	if err != nil {
		onInternalError(err)
		return
	}

	for name, value := range jsonAPI.Headers {
		request.Header.Add(name, value)
	}
	response, err := client.Do(request)
	if err != nil {
		onInternalError(err)
		return
	}

	if response.StatusCode >= 300 {
		handleHTTPError(response, onHTTPError, onInternalError)
		return
	}

	handleSuccess(response, responseBody, onSuccess, onInternalError)
}

// Get request
func (jsonAPI *JSONAPI) Get(url string, parameters url.Values,
	responseBody interface{}, onSuccess SuccessCallback, onHTTPError HTTPErrorCallback,
	onInternalError InternalErrorCallback) {
	jsonAPI.request("GET", url, parameters, nil, responseBody, onSuccess,
		onHTTPError, onInternalError)
}

// Put request
func (jsonAPI *JSONAPI) Put(url string, parameters url.Values,
	requestBody interface{}, responseBody interface{}, onSuccess SuccessCallback,
	onHTTPError HTTPErrorCallback, onInternalError InternalErrorCallback) {
	jsonAPI.request("PUT", url, parameters, requestBody, responseBody, onSuccess,
		onHTTPError, onInternalError)
}

// Post request
func (jsonAPI *JSONAPI) Post(url string, parameters url.Values,
	requestBody interface{}, responseBody interface{}, onSuccess SuccessCallback,
	onHTTPError HTTPErrorCallback, onInternalError InternalErrorCallback) {
	jsonAPI.request("POST", url, parameters, requestBody, responseBody, onSuccess,
		onHTTPError, onInternalError)
}

// Delete request
func (jsonAPI *JSONAPI) Delete(url string, parameters url.Values,
	responseBody interface{}, onSuccess SuccessCallback, onHTTPError HTTPErrorCallback,
	onInternalError InternalErrorCallback) {
	jsonAPI.request("DELETE", url, parameters, nil, responseBody, onSuccess,
		onHTTPError, onInternalError)
}

func handleSuccess(response *http.Response, data interface{}, onSuccess SuccessCallback,
	onInternalError InternalErrorCallback) {
	body, err := body(response)
	if err != nil {
		onInternalError(err)
		return
	}

	if len(body) != 0 {
		err = json.Unmarshal(body, &data)
		if err != nil {
			onInternalError(err)
			return
		}
	}

	onSuccess()
}

func handleHTTPError(response *http.Response, onHTTPError HTTPErrorCallback,
	onInternalError InternalErrorCallback) {
	body, err := body(response)
	if err != nil {
		onInternalError(err)
		return
	}

	var Error Error
	err = json.Unmarshal(body, &Error)
	if err != nil {
		onInternalError(err)
		return

	}

	onHTTPError(Error.Status, Error.Message, Error.Error)
}

func body(response *http.Response) ([]byte, error) {
	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	return body, err
}
