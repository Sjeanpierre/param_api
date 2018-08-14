package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type ssmClient struct {
	client *ssm.SSM
}

type parameter struct {
	Name     string
	Versions []paramHistory
}

type parameters []parameter

type paramHistory struct {
	Value   string
	Version string
}

func NewClient(region string) *ssm.SSM {
	session := session.Must(session.NewSession())
	if DebugMode {
		session.Config.WithRegion(region).WithLogLevel(aws.LogDebugWithHTTPBody) //.WithMaxRetries(2)
	} else {
		session.Config.WithRegion(region)
	}
	return ssm.New(session)
}

var (
	CACHEREF = map[string]map[string]string{}
)

func Deserialize(encoded string) (map[string]string, error) {
	params := make(map[string]string)
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		error := fmt.Sprintf("Error decoding value returned by single key param: %s", err.Error())
		return params, errors.New(error)
	}
	gz, err := gzip.NewReader(bytes.NewBuffer(compressed))
	defer gz.Close()
	if err != nil {
		error := fmt.Sprintf("Error decompressing value returned by single key param: %s", err.Error())
		return params, errors.New(error)
	}
	jsonData, err := ioutil.ReadAll(gz)
	if err != nil {
		error := fmt.Sprintf("Error reading decompressed json stream: %s", err.Error())
		return params, errors.New(error)
	}
	err = json.Unmarshal(jsonData, &params)
	if err != nil {
		error := fmt.Sprintf("Error unmarshalling JSON to struct: %s", err.Error())
		return params, errors.New(error)
	}
	return params, nil
}

////// SINGLE KEY MODE ONLY
//////
//////
//////
func (s ssmClient) SingleParam(paramName string) map[string]string {
	empty := make(map[string]string)

	// Get requested parameter
	CACHEREF[paramName] = s.CacheRequestedParam(paramName)

	// Iterate over param data to get ssm:// references
	for i := range CACHEREF[paramName] {
		if strings.HasPrefix(CACHEREF[paramName][i], "ssm://") {
			// Trim the ssm:// off of the key so we know the full name of the param to get
			keyName := strings.Trim(CACHEREF[paramName][i], "ssm://")

			// Request param store for the key above and store it in the CACHEREF
			CACHEREF[keyName] = s.CacheRequestedParam(keyName)

			// If the reference json data has the key required, get the value
			if val, ok := CACHEREF[keyName][i]; ok {
				CACHEREF[paramName][i] = val
				fmt.Printf("Found key %s with value %s", i, val)
				// If the reference json data does not have the key we require, return empty
			} else {
				fmt.Printf("Not Found: key %s", i)
				return empty
			}
		}
	}
	return CACHEREF[paramName]
}

func (s ssmClient) CacheRequestedParam(paramName string) map[string]string {
	empty := make(map[string]string)

	if _, ok := CACHEREF[paramName]; ok {
		if DebugMode {
			fmt.Printf("%s is already cached, skipping\n", paramName)
		}
		return CACHEREF[paramName]
	}

	pi := &ssm.GetParameterInput{Name: aws.String(paramName),
		WithDecryption: aws.Bool(true)}
	r, err := s.client.GetParameter(pi)
	if err != nil {
		fmt.Println(err.Error())
		return empty
	}
	ret, err := Deserialize(*r.Parameter.Value)
	if err != nil {
		fmt.Println(err)
		return empty
	}

	return ret
}

////// ANYTHING BEYOND THIS POINT IS FOR NON-SINGLE KEY MODE ONLY
//////
//////
//////
func (s ssmClient) WithPrefix(prefix string) parameters {
	var names parameters
	resp, err := s.ParamList(prefix)
	if err != nil {
		log.Println("Encountered an error listing params", err)
		return parameters{}
	}
	for _, param := range resp.Parameters {
		names = append(names, parameter{*param.Name, []paramHistory{}})
	}
	return names
}

func (s ssmClient) ParamList(filter string) (*ssm.DescribeParametersOutput, error) {
	//limit of 50 parameters, unless extra logic is added to paginate
	params := &ssm.DescribeParametersInput{
		MaxResults: aws.Int64(50),
		Filters: []*ssm.ParametersFilter{
			{
				Values: []*string{
					aws.String(filter),
				},
				Key: aws.String("Name"),
			},
		},
	}
	return s.client.DescribeParameters(params)
}

func (p parameters) IncludeHistory(s ssmClient) parameters {
	var params parameters
	for _, param := range p {
		param.history(s)
		params = append(params, param)
	}
	return params
}

func (p *parameter) history(s ssmClient) {
	//todo, return error
	pi := &ssm.GetParametersInput{
		Names:          []*string{&p.Name},
		WithDecryption: aws.Bool(true),
	}
	hpi := &ssm.GetParameterHistoryInput{
		Name:           &p.Name,
		WithDecryption: aws.Bool(true),
	}
	resp, err := s.client.GetParameterHistory(hpi)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	r, err := s.client.GetParameters(pi)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	re, err := s.ParamList(p.Name)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//todo, guard against empty param
	//this is being done in order to get the current version description
	p.Versions = append(p.Versions, paramHistory{Value: *r.Parameters[0].Value, Version: *re.Parameters[0].Description})
	var hist []paramHistory
	var des string
	for _, param := range resp.Parameters {
		if param.Description != nil {
			des = *param.Description

		}
		val := *param.Value
		hist = append(hist, paramHistory{Value: val, Version: des})
	}
	p.Versions = append(p.Versions, hist...)
	return
}

func (p parameters) withVersion(version string) map[string]string {
	paramsDoc := make(map[string]string)
	//todo, deserialize right here

	for _, param := range p {
		ver, err := param.containsVersion(version)
		if err != nil {
			log.Printf("Error: could not find version: %v for param %v", version, param.Name)
			continue
		}
		if SingleKeyMode {
			decodedData, err := Deserialize(ver.Value)
			if err != nil {
				log.Printf("Could not retrieve single key param: %s", err.Error())
				continue
			}
			return decodedData
		}
		ParsedName := strings.Split(param.Name, ".") //todo, check if envName matches ENV VAR regex
		envName := ParsedName[len(ParsedName)-1]
		paramsDoc[envName] = ver.Value
	}
	return paramsDoc
}

func (p parameter) containsVersion(version string) (paramHistory, error) {
	for _, v := range p.Versions {
		if v.Version == version {
			return v, nil
		}
	}
	return paramHistory{}, errors.New("could not find version")
}
