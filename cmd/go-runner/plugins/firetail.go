package plugins

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"io"
	"os"
	"strings"

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

        mockRequest := httptest.NewRequest(
                req.Method(), string(req.Path()),
                io.NopCloser(bytes.NewBuffer(body)))

	headers := req.Header().View()
        for k, v := range headers {
                // convert value (v) to comma-delimited values
                // key "k" is still as it is
                mockRequest.Header.Add(k, v[0])
        }

	// Serve the request to the middlware
	myMiddleware.ServeHTTP(localResponseWriter, mockRequest)

	middlewareResponseBodyBytes, err := io.ReadAll(localResponseWriter.Body)

	if err != nil {
		log.Errorf("Failed to read request body bytes from middleware, err: ", err.Error())

                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
                }
	}

	if string(middlewareResponseBodyBytes) != string(placeholderResponse) {
		log.Errorf("Middleware altered response body, original: %s, new: %s", string(placeholderResponse), string(middlewareResponseBodyBytes))

                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
                }
	}
}

func (p *Firetail) ResponseFilter(conf interface{}, res pkgHTTP.Response) {
        firetailApiToken := os.Getenv("FIRETAIL_API_KEY")
        firetailUrl := os.Getenv("FIRETAIL_URL")

        firetailApiToken = strings.TrimSpace(firetailApiToken)
	firetailUrl      = strings.TrimSpace(firetailUrl)

        firetailMiddleware, err := firetail.GetMiddleware(&firetail.Options{
                OpenapiSpecPath:          "./appspec.yml",
                LogsApiToken:             firetailApiToken,
                LogsApiUrl:               firetailUrl,
                DebugErrs:                true,
                EnableRequestValidation:  false,
                EnableResponseValidation: true,
        })

        if err != nil {
                log.Errorf("Failed to initialise Firetail middleware, err:", err.Error())
        }

	resBody, err := res.ReadBody()

	// most request stuff is missing here
	// need to find a way to get data from `pkgHTTP.Request`
	// such as method/protocol/SrcIP(),req.Header() and Body from RequestFilter()

	resource, err := res.Var("request_uri")
	if err != nil {
                log.Errorf("Error getting request uri")
	}

        method, err := res.Var("request_method")
        if err != nil {
                log.Errorf("Error getting request method")
        }

        reqBody, err := res.Var("request_body")
        if err != nil {
                log.Errorf("Error getting request body")
        }

	/*

	ip, err := res.Var("remote_addr")
	if err != nil {
                log.Errorf("Error getting remote address")
	} 

	protocol, err := res.Var("server_protocol")
	if err != nil {
                log.Errorf("Error getting server protocol")
	}

	scheme, err := res.Var("scheme")
	if err != nil {
                log.Errorf("Error getting scheme")
	}

	host, err := res.Var("http_host")
	if err != nil {
	        log.Errorf("Error getting http host")
	} */

        // Create a fake handler
        myHandler := &stubHandler{
                responseCode:  res.StatusCode(),
                responseBytes: resBody,
        }

	// Create our middleware instance with the stub handler
	myMiddleware := firetailMiddleware(myHandler)

	// Create a local response writer to record what the middleware says we should respond with
	localResponseWriter := httptest.NewRecorder()

        log.Infof("Firetail URL: %s", firetailUrl)

	// Serve the request to the middlware
	myMiddleware.ServeHTTP(localResponseWriter, httptest.NewRequest(
		string(method), string(resource),
                io.NopCloser(bytes.NewBuffer(reqBody)),
	))

	middlewareResponseBodyBytes, err := io.ReadAll(localResponseWriter.Body)

	if err != nil {
		log.Errorf("Failed to read response body bytes from middleware, err:", err.Error())

                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
                }
	}

	if localResponseWriter.Code != res.StatusCode() {
		//log.Errorf("Middleware altered status code from %d to %d", res.StatusCode, localResponseWriter.Code)

                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
		}
	}

	if string(middlewareResponseBodyBytes) != string(resBody) {
		//log.Errorf("Middleware altered response body, original: %s, new: %s", string(body), string(middlewareResponseBodyBytes))

                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
                }
	}

	if err != nil {
		log.Errorf("Failed to read request body bytes from middleware, err: ", err.Error())

                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
                }
	}

	if string(middlewareResponseBodyBytes) != string(resBody) {
		log.Errorf("Middleware altered response body, original: %s, new: %s", string(resBody), string(middlewareResponseBodyBytes))

                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
                }
	}
}

func init() {
  err := plugin.RegisterPlugin(&Firetail{})
  if err != nil {
    log.Fatalf("failed to register plugin firetail: %s", err)
  }
}
