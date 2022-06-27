// Package v1 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.11.0 DO NOT EDIT.
package v1

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
)

// Cluster defines model for Cluster.
type Cluster struct {
	CouchbaseConfig struct {
		ManagementPort *float32 `json:"managementPort,omitempty"`
		Password       string   `json:"password"`
		UseTLS         *bool    `json:"useTLS,omitempty"`
		Username       string   `json:"username"`
	} `json:"couchbaseConfig"`
	Hostname      string `json:"hostname"`
	MetricsConfig *struct {
		MetricsPort *float32 `json:"metricsPort,omitempty"`
	} `json:"metricsConfig,omitempty"`
	Name *string `json:"name,omitempty"`
}

// Sgw defines model for Sgw.
type Sgw struct {
	Hostname      string `json:"hostname"`
	MetricsConfig *struct {
		MetricsPort *float32 `json:"metricsPort,omitempty"`
	} `json:"metricsConfig,omitempty"`
	Name      *string `json:"name,omitempty"`
	SgwConfig struct {
		Password string `json:"password"`
		Username string `json:"username"`
	} `json:"sgwConfig"`
}

// PostClustersAddJSONBody defines parameters for PostClustersAdd.
type PostClustersAddJSONBody = Cluster

// PostSgwAddJSONBody defines parameters for PostSgwAdd.
type PostSgwAddJSONBody = Sgw

// PostClustersAddJSONRequestBody defines body for PostClustersAdd for application/json ContentType.
type PostClustersAddJSONRequestBody = PostClustersAddJSONBody

// PostSgwAddJSONRequestBody defines body for PostSgwAdd for application/json ContentType.
type PostSgwAddJSONRequestBody = PostSgwAddJSONBody

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Add a new Couchbase cluster to Prometheus
	// (POST /clusters/add)
	PostClustersAdd(ctx echo.Context) error
	// Collects diagnostic information about CMOS for Support analysis.
	// (POST /collectInformation)
	PostCollectInformation(ctx echo.Context) error
	// Outputs the OpenAPI specification for this API.
	// (GET /openapi.json)
	GetOpenapiJson(ctx echo.Context) error
	// Add a new Sync Gateway cluster to Prometheus
	// (POST /sgw/add)
	PostSgwAdd(ctx echo.Context) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// PostClustersAdd converts echo context to params.
func (w *ServerInterfaceWrapper) PostClustersAdd(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.PostClustersAdd(ctx)
	return err
}

// PostCollectInformation converts echo context to params.
func (w *ServerInterfaceWrapper) PostCollectInformation(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.PostCollectInformation(ctx)
	return err
}

// GetOpenapiJson converts echo context to params.
func (w *ServerInterfaceWrapper) GetOpenapiJson(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetOpenapiJson(ctx)
	return err
}

// PostSgwAdd converts echo context to params.
func (w *ServerInterfaceWrapper) PostSgwAdd(ctx echo.Context) error {
	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.PostSgwAdd(ctx)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {
	RegisterHandlersWithBaseURL(router, si, "")
}

// Registers handlers, and prepends BaseURL to the paths, so that the paths
// can be served under a prefix.
func RegisterHandlersWithBaseURL(router EchoRouter, si ServerInterface, baseURL string) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.POST(baseURL+"/clusters/add", wrapper.PostClustersAdd)
	router.POST(baseURL+"/collectInformation", wrapper.PostCollectInformation)
	router.GET(baseURL+"/openapi.json", wrapper.GetOpenapiJson)
	router.POST(baseURL+"/sgw/add", wrapper.PostSgwAdd)

}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/9RWTW/jNhP+KwTf9+hISlK0WZ2apsEiRdIE1d7SPdDUSOKGIlnO0KoR+L8XpOX4I3a8",
	"WbRAexM4nI/nmWeGeubS9s4aMIS8fOYoO+hF+rzSAQl8/BR1rUhZI/SDtw48KUBeNkIjTLjbOIrhguym",
	"AuHKmka17/TuhREt9GDowXqKJzU0Imji5UXx4XTCae6Al9yEfgqeLybcCcTB+jreHY1IXpk2GgPCp9tq",
	"wzS1VoMwo80b0cMex8WEe/gjKA81Lx/XNzeyfX4pxU6/gKQYsbNIByJOeA/klcRvY2Xpu6Jkh4PFnlK+",
	"DthurzYg7MNXtUOMuV3bvx/0hGM7fFMJx7T1t+pnx2ld85tdiW7KNHY5eoaETGxBL5TmZTL9+NLmTNqe",
	"r3jiV6vjCbsxMuMTHnz06Ygclnm+7baY8BpQeuUiebzkv11Xn9jlww2zDaMO2LLa4EW0swr8TMkIWSsJ",
	"BhNNY+JLJ2QH7CwrXuUchiETyZxZ3+ajL+a3N1fXv1bXJ9Entl6R3oLA7qxRZCP97PdQFGffs/spgp+J",
	"qdKK5qwiIZ/YycEqZ+BxiWt2GjNYB0Y4xUt+nhXZeWoddUkVuVwuRsxFnaThLCbSo3RS3Jual/zBIo0r",
	"FC/rmi/bC0g/2Xq+aheY5Cmc00om3/wLxjJWmzh+/d9Dw0v+v3y9qvNxT+erJb3Y1g/5AOkAnY0ExjBn",
	"RfGutO8YFPuURGdCz8vHmHst1JeNu6Nv+3RAzdsqG/ExUddQMwxSAmITtJ6niBj6Xvh5FFVdM8EMDGwt",
	"irFRjCx78LYH6iBg8sul1Rok3ZjG+l4sk73Zydf3j9JL8CflTgtleGmC1q/AVeRB9HGAtG3bqF0byAVi",
	"jbc9G0s8UeucGXY7sMe6kNVKtMYiKck2HJiY2kDs6u6+Yo31rArOWU9MGKHnqDBbsjGqPVuJoIU9NHwE",
	"ul/e+wW/Bv7b6jra+ZgrrpfzrGDoQKpmDJaAUKcwbp8dOu4Tf5gW0irAYecRPbbD8VGu2mE1xQ7/2zNV",
	"zY1kHwXBIOaHBuuf2FXx/+HonoqDsn+ut8o+NNpRDeDjNufl4/MO8FsrhWZ3SnqrFXVbr0+Z5zqa41tb",
	"XhQXRS7TU5ELp/L0JuyP9jPMQFsXf1kPx/vh9MN3L4E+L/4KAAD//6e0iIJ1CwAA",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %s", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	var res = make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	var resolvePath = PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		var pathToFile = url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
