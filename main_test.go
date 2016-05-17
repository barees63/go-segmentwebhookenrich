package main

import (
	"appengine"
	"appengine/aetest"
	"appengine/urlfetch"
	"bytes"
	"fmt"
	"github.com/bmizerany/assert"
	"github.com/jarcoal/httpmock"
	lytics "github.com/lytics/go-lytics"
	"net/http/httptest"
	"testing"
	"time"
)

var (
	testConfig = &Config{
		lyticsAPIKey: mockLyticsKey,
		webhook:      "sparkpost",
		webhooks: map[string]map[string]string{
			"sparkpost": {
				"url":      "https://api.sparkpost.com/api/v1/transmissions",
				"apikey":   mockSpKey,
				"template": mockSpTemplate,
			},
		},
		getOptimalHour:       true,
		recommendationFilter: mockFilter,
		event: &Event{
			name:    "segment_entered",
			segment: "MockSegmentName",
		},
	}
)

func postTest(filename string, inst aetest.Instance, ctx appengine.Context) string {
	r, _ := inst.NewRequest("POST", "/post", bytes.NewReader(readMockJson(filename)))
	w := httptest.NewRecorder()
	testConfig.enrichWebhook(w, r, ctx)
	return w.Body.String()
}

func TestEnrichWebhook(t *testing.T) {
	ctx, err := aetest.NewContext(nil)
	assert.Equal(t, err, nil)
	testConfig.setClient(urlfetch.Client(ctx))
	defer ctx.Close()

	inst, err := aetest.NewInstance(nil)
	assert.Equal(t, err, nil)
	defer inst.Close()

	httpmock.ActivateNonDefault(testConfig.client)
	registerMocks()
	defer httpmock.DeactivateAndReset()

	// Test event matching
	// non-matching segment name
	assert.Equal(t, postTest("segment_webhook_seg_name", inst, ctx), `{"message":"not processed: segment name did not match","status":204}`)

	// non-matching event name
	assert.Equal(t, postTest("segment_webhook_evt_name", inst, ctx), `{"message":"not processed: event name did not match","status":204}`)

	fmt.Println("+++ event matching error handling verified")

	// Test recommendation handling
	// bad Lytics API key
	testConfig.lyticsAPIKey = "BadMockKey"
	assert.Equal(t, postTest("segment_webhook_valid", inst, ctx), `{"message":"could not get recommendation for this user","status":500}`)
	testConfig.lyticsAPIKey = mockLyticsKey

	// bad filter
	testConfig.recommendationFilter = "FILTER AND (global.badTopic > 0)"
	assert.Equal(t, postTest("segment_webhook_valid", inst, ctx), `{"message":"could not get recommendation for this user","status":500}`)
	testConfig.recommendationFilter = mockFilter

	fmt.Println("+++ lytics content recommendation error handling verified")

	webhook := testConfig.webhooks[testConfig.webhook]

	// Test Sparkpost deploy  - REMOVE IF USING CUSTOM WEBHOOK
	// bad Sparkpost API key
	webhook["apikey"] = "BadSpMockKey"
	assert.Equal(t, postTest("segment_webhook_valid", inst, ctx), `{"message":"could not send webhook","status":500}`)
	webhook["apikey"] = mockSpKey

	// bad template name
	webhook["template"] = "bad-template"
	assert.Equal(t, postTest("segment_webhook_valid", inst, ctx), `{"message":"could not send webhook","status":500}`)
	webhook["template"] = mockSpTemplate

	fmt.Println("+++ sparkpost email error handling verified")

	// success
	assert.Equal(t, postTest("segment_webhook_valid", inst, ctx), `{"message":"success","status":200}`)
}

func TestPrepPayload(t *testing.T) {
	evt := &SegmentEvent{
		Properties: map[string]interface{}{
			"email": "test@example.com",
			"hourly": map[string]interface{}{
				"7": float64(1),
			},
		},
	}

	data := lytics.Recommendation{
		Document: &lytics.Document{
			Url:         "www.example.com/blog/post/1",
			Title:       "Title of Blog Post",
			Description: "this is a mock blog post",
		},
		Confidence: 0.7654,
		Visited:    false,
	}

	payload := testConfig.PrepPayload(evt, data)

	recipients, _ := payload["recipients"].([]map[string]interface{})
	assert.Equal(t, recipients[0]["address"], evt.Properties["email"])

	substitution, _ := recipients[0]["substitution_data"].(map[string]interface{})
	assert.Equal(t, substitution["data"], data)

	content, _ := payload["content"].(map[string]string)
	assert.Equal(t, content["template_id"], testConfig.webhooks[testConfig.webhook]["template"])

	options, ok := payload["options"].(map[string]interface{})
	if time.Now().Hour() != 7 {
		assert.Equal(t, options["start_time"], evt.SendTime().Format(time.RFC3339))
	} else {
		assert.Equal(t, ok, false)
	}

	fmt.Println("+++ PrepPayload verified")
}

func TestMakeRequest(t *testing.T) {
	ctx, err := aetest.NewContext(nil)
	assert.Equal(t, err, nil)
	testConfig.setClient(urlfetch.Client(ctx))
	defer ctx.Close()

	httpmock.ActivateNonDefault(testConfig.client)
	registerMocks()
	defer httpmock.DeactivateAndReset()

	payload := map[string]interface{}{
		"recipients": []map[string]interface{}{
			map[string]interface{}{
				"address": "test@example.com",
			},
		},
		"content": map[string]string{
			"template_id": testConfig.webhooks[testConfig.webhook]["template"],
		},
	}

	webhook := testConfig.webhooks[testConfig.webhook]

	webhook["apikey"] = "BadSpMockKey"
	err = testConfig.MakeRequest(payload)
	assert.NotEqual(t, err, nil)
	webhook["apikey"] = mockSpKey

	err = testConfig.MakeRequest(payload)
	assert.Equal(t, err, nil)
}

func TestSendTime(t *testing.T) {
	evt := &SegmentEvent{
		Properties: map[string]interface{}{
			"hourly": map[string]interface{}{
				"0":  float64(583),
				"1":  float64(414),
				"14": float64(1),
				"17": float64(721),
				"18": float64(1140),
				"23": float64(1138),
			},
		},
	}

	sendTime := evt.SendTime()
	now := time.Now()

	// current hour is not optimal
	if now.Hour() != 18 {
		assert.Equal(t, sendTime.Hour(), 18)
		assert.T(t, sendTime.After(now))
		dur := sendTime.Sub(now)
		assert.T(t, dur.Hours() < 24)

		// current hour matches optimal
	} else {
		assert.Equal(t, sendTime, nil)
	}

	fmt.Println("+++ getOptimalSendTime verified")
}
