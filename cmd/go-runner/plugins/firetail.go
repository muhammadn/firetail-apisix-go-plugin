package plugins

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"io"

	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
	"github.com/apache/apisix-go-plugin-runner/pkg/plugin"
	"bytes"

	firetail "github.com/FireTail-io/firetail-go-lib/middlewares/http"
)

type Firetail struct {
	plugin.DefaultPlugin
}

type FiretailConf struct {
  Body string `json:"body"`
}

func (p *Firetail) Name() string {
  return "firetail"
}

func (p *Firetail) ParseConf(in []byte) (interface{}, error) {
        conf := FiretailConf{}
        err := json.Unmarshal(in, &conf)
        return conf, err
}

func (p *Firetail) RequestFilter(conf interface{}, res http.ResponseWriter, req pkgHTTP.Request) {
	firetailMiddleware, err := firetail.GetMiddleware(&firetail.Options{
		OpenapiSpecPath:          "./appspec.yml",
		LogsApiToken:             "",
		LogsApiUrl:               "",
		DebugErrs:                true,
		EnableRequestValidation:  true,
		EnableResponseValidation: false,
	})

	if err != nil {
		log.Errorf("Failed to initialise Firetail middleware, err:", err.Error())
	}

	// Create a fake handler
	placeholderResponse := []byte{}
	myHandler := &stubHandler{
		responseCode:  200,
		responseBytes: placeholderResponse,
	}

	// Create our middleware instance with the stub handler
	myMiddleware := firetailMiddleware(myHandler)

	localResponseWriter := httptest.NewRecorder()

	body, err := req.Body()
	if err != nil {
                log.Errorf("cannot get body:", err.Error())
	}

	// Serve the request to the middlware
	myMiddleware.ServeHTTP(localResponseWriter, httptest.NewRequest(
		req.Method(), string(req.Path()),
		io.NopCloser(bytes.NewBuffer(body)),
	))

	middlewareResponseBodyBytes, err := io.ReadAll(localResponseWriter.Body)

	res.Header().Add("X-Resp-A6-Runner", "Go")
	_, err = res.Write(middlewareResponseBodyBytes)
	if err != nil {
		log.Errorf("failed to write %s", err)
	}
}

func init() {
  err := plugin.RegisterPlugin(&Firetail{})
  if err != nil {
    log.Fatalf("failed to register plugin firetail: %s", err)
  }
}
