package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type Response struct {
	status  int
	Message string
	Data    interface{}
}

type paramRequest struct {
	Application string
	Environment string
	Version     string
	Landscape   string
}

func (p paramRequest) valid() bool {
	if p.Application == "" || p.Environment == "" || p.Version == "" || p.Landscape == "" {
		return false
	}
	return true
}

func (p paramRequest) envPrefix() string {
	return fmt.Sprintf("%s.%s.%s", p.Landscape, p.Environment, p.Application)
}

func (p paramRequest) cacheKey() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(p.identifier())))
}

func (p paramRequest) identifier() string {
	return fmt.Sprintf("%s@%s", p.envPrefix(), p.Version)
}

var (
	DebugMode     = false
	SingleKeyMode = false
	CACHE         = make(map[string]Response)
	region        = os.Getenv("AWS_REGION")
	debug         = os.Getenv("DEBUG")
	SingleKey     = os.Getenv("SINGLE_KEY_MODE")
)

func main() {
	api()
}

func api() {
	router := mux.NewRouter().StrictSlash(true)
	registerHandlers(router)
	loggedRouter := handlers.LoggingHandler(os.Stdout, router)
	log.Println("Validating Config") //todo, validate config
	if region == "" {
		log.Fatal("Environment variable AWS_REGION undefined")
		//todo, check against list of known regions
	}
	//in debug mode no caching takes place
	//logs are produced in greater detail
	if debug != "" {
		log.Printf("DEBUG flag set to %+v - attempting to parse to boolean", debug)
		debugenabled, err := strconv.ParseBool(debug)
		if err != nil {
			log.Printf("Warning: Could not parse debug flag, value provided was %s\n %s", DebugMode, err.Error())
			log.Println("debug mode: false")
			DebugMode = false
		} else {
			DebugMode = debugenabled
			log.Printf("debug mode set to %+v", DebugMode)
		}
	}
	if SingleKey != "" {
		sk, err := strconv.ParseBool(SingleKey)
		if err != nil {
			log.Fatalf("Could not start application, unknown value '%v' set "+
				"for SINGLE_KEY_MODE ENV VAR - true or false required", SingleKey)
		}
		SingleKeyMode = sk
	}
	log.Println("Started: Ready to serve")
	log.Fatal(http.ListenAndServe(":8080", loggedRouter)) //todo, refactor to make port dynamic
}

func registerHandlers(r *mux.Router) {
	r.NotFoundHandler = http.HandlerFunc(notFoundHandler)
	r.HandleFunc("/params", envHandler).Methods("POST")
	r.HandleFunc("/service/health", healthHandler).Methods("GET")
}

func parseParamRequestBody(b io.ReadCloser) paramRequest {
	decoder := json.NewDecoder(b)
	var p paramRequest
	err := decoder.Decode(&p)
	if err != nil {
		log.Printf("encountered issue decoding request body; %s", err.Error())
		return paramRequest{}
	}
	return p
}

func (p paramRequest) getData() map[string]string {
	c := ssmClient{NewClient(region)}
	if SingleKeyMode {
		//paramName := c.WithPrefix(fmt.Sprintf("%s.%s.%s.%s", p.Landscape, p.Environment, p.Application,p.Version))
		//return paramName.IncludeHistory(c).withVersion(p.Version)
		return c.SingleParam(fmt.Sprintf("%s.%s.%s.%s", p.Landscape, p.Environment, p.Application,p.Version))

	}
	paramNames := c.WithPrefix(p.envPrefix()) //todo, provide the full known param, is composite key
	return paramNames.IncludeHistory(c).withVersion(p.Version) //todo, return error
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var m = make(map[string]string)

	m["region"] = region
	m["single_key"] = fmt.Sprintf("%t", SingleKeyMode)
	m["debug"] = fmt.Sprintf("%t", DebugMode)

	resp := Response{status: http.StatusOK, Data: m} //todo, check length of list before returning

	JSONResponseHandler(w, resp)
}

func envHandler(w http.ResponseWriter, r *http.Request) {
	p := parseParamRequestBody(r.Body)
	if !p.valid() {
		badRequest(w, p)
		return
	}
	log.Printf("Processing request for %s uniquely identified as %+v", p.identifier(), p.cacheKey())
	cached, ok := CACHE[p.cacheKey()]
	if ok {
		if DebugMode {
			log.Println("Bypassing response cache due to debug mode")
		} else {
			log.Printf("Retrieved parameters from cache")
			JSONResponseHandler(w, cached)
			return

		}
	}
	data := p.getData()
	resp := Response{status: http.StatusOK, Data: data} //todo, check length of list before returning
	//only cache data when elements were found
	//possible bug - existing versions where new elements are added will still return cached data
	//should not be a problem since container will be restarted upon config changes
	//latest is treated as a special version indicator which should not be cached
	if len(data) > 0 && p.Version != "latest" {
		if DebugMode {
			log.Println("Skipping response caching due to debug mode")
		} else {
			log.Println("Caching result set")
			CACHE[p.cacheKey()] = resp
		}
	}
	JSONResponseHandler(w, resp)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var m = make(map[string]string)
	m["error"] = fmt.Sprintf("Route %s not found with method %s, please check request and try again",
		r.URL.Path, r.Method)
	resp := Response{Data: m, status: http.StatusNotFound}
	JSONResponseHandler(w, resp)
}

func badRequest(w http.ResponseWriter, p paramRequest) {
	w.Header().Set("Content-Type", "application/json")
	var m = make(map[string]string)
	expected := strings.ToLower(fmt.Sprintf("%+v", paramRequest{"STRING", "STRING", "STRING", "STRING"}))
	m["error"] = fmt.Sprintf("Bad request, expected: %s, got: %s", expected, strings.ToLower(fmt.Sprintf("%+v", p)))
	resp := Response{Data: m, status: http.StatusBadRequest}
	JSONResponseHandler(w, resp)
}

func JSONResponseHandler(w http.ResponseWriter, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.status)
	json.NewEncoder(w).Encode(resp.Data)
}
