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

type MiddlewareFunction func(r *http.Request) error

// JSONAPI struct
type JSONAPI struct {
	BaseURL string
	Headers map[string]string

	middleware []MiddlewareFunction
}

// SuccessCallback runs on a successfull request and parse
type SuccessCallback func()

// HTTPErrorCallback runs on a errored HTTP request
type HTTPErrorCallback func(statusCode int, statusMessage, errorMessage string)

// InternalErrorCallback runs on an internal error
type InternalErrorCallback func(error)

var client = &http.Client{}

type Request struct {
	api     *JSONAPI
	request *http.Request
}

func (r *Request) SetHeader(key, value string) *Request {
	r.request.Header.Set(key, value)
	return r
}

func (r *Request) execute(verb, urlString string, parameters url.Values,
	requestBody interface{}, responseBody interface{}, onSuccess SuccessCallback,
	onHTTPError HTTPErrorCallback, onInternalError InternalErrorCallback) (response *http.Response, err error) {
	r.request.Method = verb
	r.request.URL, err = url.Parse(r.api.BaseURL + urlString + "?" + parameters.Encode())
	if err != nil {
		return
	}

	if requestBody != nil {
		var serializedRequestBody []byte
		serializedRequestBody, err = json.Marshal(requestBody)
		if err != nil {
			onInternalError(err)
			return
		}

		r.request.Body = ioutil.NopCloser(bytes.NewReader(serializedRequestBody))
	}

	if err != nil {
		onInternalError(err)
		return
	}

	for name, value := range r.api.Headers {
		r.request.Header.Add(name, value)
	}

	for _, middleware := range r.api.middleware {
		err = middleware(r.request)
		if err != nil {
			return
		}
	}

	response, err = client.Do(r.request)
	if err != nil {
		onInternalError(err)
		return
	}

	if response.StatusCode >= 300 {
		handleHTTPError(response, onHTTPError, onInternalError)
		return
	}

	handleSuccess(response, responseBody, onSuccess, onInternalError)

	return
}

// Get request
func (r *Request) Get(url string, parameters url.Values,
	responseBody interface{}, onSuccess SuccessCallback, onHTTPError HTTPErrorCallback,
	onInternalError InternalErrorCallback) (response *http.Response, err error) {
	return r.execute("GET", url, parameters, nil, responseBody, onSuccess,
		onHTTPError, onInternalError)
}

// Put request
func (r *Request) Put(url string, parameters url.Values,
	responseBody interface{}, requestBody interface{}, onSuccess SuccessCallback, onHTTPError HTTPErrorCallback,
	onInternalError InternalErrorCallback) (response *http.Response, err error) {
	return r.execute("PUT", url, parameters, requestBody, responseBody, onSuccess,
		onHTTPError, onInternalError)
}

// Post request
func (r *Request) Post(url string, parameters url.Values,
	responseBody interface{}, requestBody interface{}, onSuccess SuccessCallback, onHTTPError HTTPErrorCallback,
	onInternalError InternalErrorCallback) (response *http.Response, err error) {
	return r.execute("POST", url, parameters, requestBody, responseBody, onSuccess,
		onHTTPError, onInternalError)
}

// Delete request
func (r *Request) Delete(url string, parameters url.Values,
	responseBody interface{}, onSuccess SuccessCallback, onHTTPError HTTPErrorCallback,
	onInternalError InternalErrorCallback) (response *http.Response, err error) {
	return r.execute("DELETE", url, parameters, nil, responseBody, onSuccess,
		onHTTPError, onInternalError)
}

func (jsonAPI *JSONAPI) R() *Request {
	r := &Request{
		api: jsonAPI,
	}

	request, _ := http.NewRequest("GET", "", nil)
	r.request = request

	return r
}

func (jsonAPI *JSONAPI) Use(mw ...MiddlewareFunction) {
	jsonAPI.middleware = append(jsonAPI.middleware, mw...)
}

// Get request
func (jsonAPI *JSONAPI) Get(url string, parameters url.Values,
	responseBody interface{}, onSuccess SuccessCallback, onHTTPError HTTPErrorCallback,
	onInternalError InternalErrorCallback) (response *http.Response, err error) {
	return jsonAPI.R().execute("GET", url, parameters, nil, responseBody, onSuccess,
		onHTTPError, onInternalError)
}

// Put request
func (jsonAPI *JSONAPI) Put(url string, parameters url.Values,
	requestBody interface{}, responseBody interface{}, onSuccess SuccessCallback,
	onHTTPError HTTPErrorCallback, onInternalError InternalErrorCallback) (response *http.Response, err error) {
	return jsonAPI.R().execute("PUT", url, parameters, requestBody, responseBody, onSuccess,
		onHTTPError, onInternalError)
}

// Post request
func (jsonAPI *JSONAPI) Post(url string, parameters url.Values,
	requestBody interface{}, responseBody interface{}, onSuccess SuccessCallback,
	onHTTPError HTTPErrorCallback, onInternalError InternalErrorCallback) (response *http.Response, err error) {
	return jsonAPI.R().execute("POST", url, parameters, requestBody, responseBody, onSuccess,
		onHTTPError, onInternalError)
}

// Delete request
func (jsonAPI *JSONAPI) Delete(url string, parameters url.Values,
	responseBody interface{}, onSuccess SuccessCallback, onHTTPError HTTPErrorCallback,
	onInternalError InternalErrorCallback) (response *http.Response, err error) {
	return jsonAPI.R().execute("DELETE", url, parameters, nil, responseBody, onSuccess,
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
	Error.Status = response.StatusCode
	Error.Message = string(body[:])
	Error.Error = response.Status
	json.Unmarshal(body, &Error)
	onHTTPError(Error.Status, Error.Message, Error.Error)
}

func body(response *http.Response) ([]byte, error) {
	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	return body, err
}
