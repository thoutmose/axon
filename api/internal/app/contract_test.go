package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/thoutmose/axon/api/openapi"
)

// Contract tests: every request the integration tests make, and every
// response they get, is checked against openapi.yaml.

var (
	specOnce   sync.Once
	spec       *openapi3.T
	specRouter routers.Router
	covered    sync.Map // operationId -> true
)

func loadSpec(t *testing.T) {
	t.Helper()
	specOnce.Do(func() {
		openapi3filter.RegisterBodyDecoder("application/zip", openapi3filter.FileBodyDecoder)
		openapi3filter.RegisterBodyDecoder("application/yaml", openapi3filter.FileBodyDecoder)
		doc, err := openapi3.NewLoader().LoadFromData(openapi.Spec)
		if err != nil {
			panic(fmt.Sprintf("openapi.yaml does not parse: %v", err))
		}
		if err := doc.Validate(context.Background()); err != nil {
			panic(fmt.Sprintf("openapi.yaml is invalid: %v", err))
		}
		// Match requests as the browser sends them: http://<host>/api/...
		doc.Servers = openapi3.Servers{{URL: "http://localhost/api"}}
		spec = doc
		if specRouter, err = gorillamux.NewRouter(doc); err != nil {
			panic(err)
		}
	})
}

func TestSpecIsValid(t *testing.T) { loadSpec(t) }

// checkContract validates one exchange. Requests are only validated when the
// server accepted them: tests send malformed requests on purpose.
func checkContract(t *testing.T, req *http.Request, reqBody []byte, res *http.Response, resBody []byte) {
	t.Helper()
	loadSpec(t)
	u, _ := url.Parse("http://localhost/api" + req.URL.Path)
	u.RawQuery = req.URL.RawQuery
	vreq := req.Clone(context.Background())
	vreq.URL = u
	vreq.Host = "localhost"
	vreq.Body = io.NopCloser(bytes.NewReader(reqBody))

	route, params, err := specRouter.FindRoute(vreq)
	if err != nil {
		t.Errorf("contract: %s %s is not documented in openapi.yaml", req.Method, req.URL.Path)
		return
	}
	covered.Store(route.Operation.OperationID, true)
	in := &openapi3filter.RequestValidationInput{
		Request: vreq, PathParams: params, Route: route,
		Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
	}
	if res.StatusCode < 400 {
		if err := openapi3filter.ValidateRequest(context.Background(), in); err != nil {
			t.Errorf("contract: request %s %s: %v", req.Method, req.URL.Path, err)
		}
	}
	out := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: in, Status: res.StatusCode, Header: res.Header,
		Body:    io.NopCloser(bytes.NewReader(resBody)),
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	}
	if err := openapi3filter.ValidateResponse(context.Background(), out); err != nil {
		t.Errorf("contract: response %d to %s %s: %v\n%s", res.StatusCode, req.Method, req.URL.Path, err, resBody)
	}
}
