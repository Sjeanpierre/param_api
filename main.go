package main

import (
	"log"
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	"os"
	"net/http"
	"encoding/json"
	"fmt"
        "crypto/sha256"
	"strings"
	"io"
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
	return fmt.Sprintf("%s@%s", p.envPrefix(),p.Version)
}

var (
	CACHE = make(map[string]Response)
	region = os.Getenv("AWS_REGION")
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
	log.Println("Started: Ready to serve")
	log.Fatal(http.ListenAndServe(":8080", loggedRouter)) //todo, refactor to make port dynamic
}

func registerHandlers(r *mux.Router) {
	r.NotFoundHandler = http.HandlerFunc(notFoundHandler)
	r.HandleFunc("/params", envHandler).Methods("POST")

}

func parseParamRequestBody(b io.ReadCloser)  paramRequest{
	decoder := json.NewDecoder(b)
	var p paramRequest
	err := decoder.Decode(&p)
	if err != nil {
		log.Printf("encountered issue decoding request body; %s",err.Error())
		return paramRequest{}
	}
	return  p
}

func (p paramRequest) getData()  map[string]string{
	c := ssmClient{NewClient(region)}
	paramNames := c.WithPrefix(p.envPrefix())
	return paramNames.IncludeHistory(c).withVersion(p.Version) //todo, return error
}

func envHandler(w http.ResponseWriter, r *http.Request) {
	p := parseParamRequestBody(r.Body)
	if !p.valid() {
		badRequest(w, p)
		return
	}
	log.Printf("Processing request for %s uniquely identified as %+v", p.identifier(),p.cacheKey())
	cached,ok := CACHE[p.cacheKey()]
	if ok {
		log.Printf("Retrieved parameters from cache")
		JSONResponseHandler(w, cached)
		return
	}
	data := p.getData()
	resp := Response{status:http.StatusOK, Data:data} //todo, check length of list before returning
	//only cache data when elements were found
	//possible bug - existing versions where new elements are added will still return cached data
	//should not be a problem since container will be restarted upon config changes
	if len(data) > 0 {
		CACHE[p.cacheKey()] = resp
	}
	JSONResponseHandler(w, resp)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var m = make(map[string]string)
	m["error"] = fmt.Sprintf("Route %s not found with method %s, please check request and try again",
		r.URL.Path,r.Method)
	resp := Response{Data:m, status:http.StatusNotFound}
	JSONResponseHandler(w, resp)
}

func badRequest(w http.ResponseWriter, p paramRequest) {
	w.Header().Set("Content-Type", "application/json")
	var m = make(map[string]string)
	expected := strings.ToLower(fmt.Sprintf("%+v", paramRequest{"STRING", "STRING", "STRING", "STRING"}))
	m["error"] = fmt.Sprintf("Bad request, expected: %s, got: %s", expected, strings.ToLower(fmt.Sprintf("%+v", p)))
	resp := Response{Data:m, status:http.StatusBadRequest}
	JSONResponseHandler(w, resp)
}

func JSONResponseHandler(w http.ResponseWriter, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.status)
	json.NewEncoder(w).Encode(resp.Data)
}