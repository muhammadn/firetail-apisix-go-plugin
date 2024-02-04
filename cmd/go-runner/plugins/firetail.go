package plugins

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"io"
	"time"
	"os"

	pkgHTTP "github.com/apache/apisix-go-plugin-runner/pkg/http"
	"github.com/apache/apisix-go-plugin-runner/pkg/log"
	"github.com/apache/apisix-go-plugin-runner/pkg/plugin"
	"bytes"

	firetail "github.com/FireTail-io/firetail-go-lib/middlewares/http"
)

// We should be using res.Header() and res.Header() instead here
// temporarily to test this out we set this placeholder
type PlaceHolderHeaders struct {}

type FiretailRequest struct {
  Ip string `json:"ip"`
  HttpProtocol string `json:"httpProtocol"`
  Uri string `json:"uri"`
  Resource string `json:"resource"`
  Method string `json:"method"`
  Headers PlaceHolderHeaders `json:"headers"`
  Body string `json:"body"`
}

type FiretailResponse struct {
  StatusCode int `json:"statusCode"`
  Body string `json:"body"`
  Headers PlaceHolderHeaders `json:"headers"`
}

type FiretailPayload struct {
  Version string `json:"version"`
  DateCreated int64 `json:"dateCreated"`
  ExecutionTime int64 `json:"executionTime"`
  Request FiretailRequest `json:"request"`
  Response FiretailResponse `json:"response"`
}

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

	if err != nil {
		//log.Errorf("Failed to read request body bytes from middleware, err: ", err.Error())

                res.Header().Add("X-Resp-A6-Runner", "Go")
                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
                }
	}

	if string(middlewareResponseBodyBytes) != string(placeholderResponse) {
		//log.Errorf("Middleware altered response body, original: %s, new: %s", string(placeholderResponse), string(middlewareResponseBodyBytes))

                res.Header().Add("X-Resp-A6-Runner", "Go")
                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
                }
	}
}

func (p *Firetail) ResponseFilter(conf interface{}, res pkgHTTP.Response) {
        firetailMiddleware, err := firetail.GetMiddleware(&firetail.Options{
                OpenapiSpecPath:          "./appspec.yml",
                LogsApiToken:             "",
                LogsApiUrl:               "",
                DebugErrs:                true,
                EnableRequestValidation:  false,
                EnableResponseValidation: true,
        })

        if err != nil {
                log.Errorf("Failed to initialise Firetail middleware, err:", err.Error())
        }

	body, err := res.ReadBody()

	// most request stuff is missing here
	// need to find a way to get data from `pkgHTTP.Request`
	// such as method/protocol/SrcIP(),req.Header() and Body from RequestFilter()
        payload := FiretailPayload{
                Version: "1.0.0-alpha",
                DateCreated: time.Now().UTC().Unix(),
                ExecutionTime: time.Now().UTC().Unix(),
                Request: FiretailRequest{
                  Ip: "127.0.0.1",
                  HttpProtocol: "HTTP/2",
                  Uri: "https://localhost/test",
                  Resource: "/test",
                  Method: "GET",
                  Headers: PlaceHolderHeaders{},
                  Body: "",
                },
                Response: FiretailResponse{
                  StatusCode: res.StatusCode(),
                  Body: string(body),
                  Headers: PlaceHolderHeaders{},
                },
        }

        firetailApiToken := os.Getenv("FIRETAIL_API_KEY")
	firetailUrl := os.Getenv("FIRETAIL_URL")

	log.Infof("Firetail URL: %s", firetailUrl)

        jsonPayload, err := json.Marshal(payload)
        if err != nil {
                log.Errorf("Error encoding json payload")
        }

        httpReq, err := http.NewRequest("POST", firetailUrl, bytes.NewBuffer(jsonPayload))
        if err != nil {
          log.Errorf("Error with parsing HTTP %s", err)
        }

        httpReq.Header.Add("Content-Type", "application/nd-json")
        httpReq.Header.Add("x-ft-api-key", firetailApiToken)

        go func() {
                client := &http.Client{}
                httpRes, err := client.Do(httpReq)

                client.Do(httpReq)
                if err != nil {
                        panic(err)
                }
                defer httpRes.Body.Close()

		// debug info
		log.Infof("HTTP status code: %d", httpRes.StatusCode)

	        b, err := io.ReadAll(httpRes.Body)
	        if err != nil {
			log.Errorf("Error reading response body: %s", err)
	        }

		log.Infof("HTTP response body: %s", string(b))
        }()

        log.Infof("payload: %s", jsonPayload)

        // Create a fake handler
        myHandler := &stubHandler{
                responseCode:  res.StatusCode(),
                responseBytes: body,
        }

	// Create our middleware instance with the stub handler
	myMiddleware := firetailMiddleware(myHandler)

	// Create a local response writer to record what the middleware says we should respond with
	localResponseWriter := httptest.NewRecorder()

	// Serve the request to the middlware
	myMiddleware.ServeHTTP(localResponseWriter, httptest.NewRequest(
		"GET", "/health",
                io.NopCloser(bytes.NewBuffer([]byte{})),
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

	if string(middlewareResponseBodyBytes) != string(body) {
		//log.Errorf("Middleware altered response body, original: %s, new: %s", string(body), string(middlewareResponseBodyBytes))

                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
                }
	}

	middlewareResponseBodyBytes, err := io.ReadAll(localResponseWriter.Body)

	res.Header().Add("X-Resp-A6-Runner", "Go")
	_, err = res.Write(middlewareResponseBodyBytes)

	if err != nil {
		log.Errorf("Failed to read request body bytes from middleware, err: ", err.Error())

                res.Header().Add("X-Resp-A6-Runner", "Go")
                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
                }
	}

	if string(middlewareResponseBodyBytes) != string(placeholderResponse) {
		log.Errorf("Middleware altered response body, original: %s, new: %s", string(placeholderResponse), string(middlewareResponseBodyBytes))

                res.Header().Add("X-Resp-A6-Runner", "Go")
                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
                }
	}
}

func (p *Firetail) ResponseFilter(conf interface{}, res pkgHTTP.Response) {
        firetailMiddleware, err := firetail.GetMiddleware(&firetail.Options{
                OpenapiSpecPath:          "./appspec.yml",
                LogsApiToken:             "",
                LogsApiUrl:               "",
                DebugErrs:                true,
                EnableRequestValidation:  false,
                EnableResponseValidation: true,
        })

        if err != nil {
                log.Errorf("Failed to initialise Firetail middleware, err:", err.Error())
        }


	body, err := res.ReadBody()

        // Create a fake handler
        myHandler := &stubHandler{
                responseCode:  res.StatusCode(),
                responseBytes: body,
        }

	// Create our middleware instance with the stub handler
	myMiddleware := firetailMiddleware(myHandler)

	// Create a local response writer to record what the middleware says we should respond with
	localResponseWriter := httptest.NewRecorder()

	// Serve the request to the middlware
	myMiddleware.ServeHTTP(localResponseWriter, httptest.NewRequest(
		method, path,
                io.NopCloser(bytes.NewBuffer([]byte{})),
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
		log.Errorf("Middleware altered status code from %d to %d", res.StatusCode, localResponseWriter.Code)

                _, err = res.Write(middlewareResponseBodyBytes)
                if err != nil {
                        log.Errorf("failed to write %s", err)
		}
	}

	if string(middlewareResponseBodyBytes) != string(body) {
		log.Errorf("Middleware altered response body, original: %s, new: %s", string(body), string(middlewareResponseBodyBytes))

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
