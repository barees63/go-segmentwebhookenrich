package main

import (
	"encoding/json"
	"fmt"
	"github.com/jarcoal/httpmock"
	"io/ioutil"
	"net/http"
)

var (
	mockLyticsKey  = "MockLyticsKey"
	mockFilter     = "FILTER AND (url LIKE \"www.example.com/blog/*\") FROM content"
	mockSpKey      = "MockSparkpostKey"
	mockSpTemplate = "MockSparkpostTemplate"
)

func readMockJson(name string) []byte {
	json, err := ioutil.ReadFile(fmt.Sprintf("mockjson/%s.json", name))
	if err != nil {
		panic(err)
	}

	return json
}

func registerMocks() {
	// Lytics Recommendation API Mock
	httpmock.RegisterResponder("GET", "https://api.lytics.io/api/content/recommend/user/emails/example@test.com",
		func(req *http.Request) (*http.Response, error) {
			queries := req.URL.Query()

			if queries.Get("key") != mockLyticsKey {
				return httpmock.NewStringResponse(401, string(readMockJson("lytics_unauthorized"))), nil
			}

			if queries.Get("ql") != mockFilter {
				return httpmock.NewStringResponse(200, string(readMockJson("lytics_invalid_filter"))), nil
			}

			return httpmock.NewStringResponse(200, string(readMockJson("lytics_recommend"))), nil
		},
	)

	// Sparkpost Transmission Mock - CHANGE/REMOVE IF USING CUSTOM WEBHOOK
	httpmock.RegisterResponder("POST", "https://api.sparkpost.com/api/v1/transmissions",
		func(req *http.Request) (*http.Response, error) {
			auth := req.Header.Get("Authorization")
			body := map[string]interface{}{}

			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				return nil, err
			}

			content, _ := body["content"].(map[string]interface{})

			if auth != mockSpKey {
				return httpmock.NewStringResponse(401, string(readMockJson("sparkpost_unauthorized"))), nil
			}

			if content["template_id"] != mockSpTemplate {
				return httpmock.NewStringResponse(422, string(readMockJson("sparkpost_invalid_template"))), nil
			}

			return httpmock.NewStringResponse(200, string(readMockJson("sparkpost_valid"))), nil
		},
	)
}
