package main

import (
	"appengine"
	"appengine/aetest"
	"bytes"
	"appengine/urlfetch"
	"fmt"
	"github.com/bmizerany/assert"
	"github.com/jarcoal/httpmock"
	"net/http/httptest"
	"testing"
	"time"
)

var (
	testConfig = &Config{
		lyticsAPIKey: mockLyticsKey,
		webhookUrl: "https://api.sparkpost.com/api/v1/transmissions",
		getOptimalHour: true,
		recommendationFilter: mockFilter,
		event: &Event{
			name: "segment_entered",
			segment: "MockSegmentName",
		},
		sparkpostTemplateId: mockSpTemplate,
		sparkpostAPIKey: mockSpKey,
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

	// Test Sparkpost deploy  - REMOVE IF USING CUSTOM WEBHOOK
	// bad Sparkpost API key
	testConfig.sparkpostAPIKey = "BadSpMockKey"
	assert.Equal(t, postTest("segment_webhook_valid", inst, ctx), `{"message":"could not send webhook","status":500}`)
	testConfig.sparkpostAPIKey = mockSpKey

	// bad template name
	testConfig.sparkpostTemplateId = "bad-template"
	assert.Equal(t, postTest("segment_webhook_valid", inst, ctx), `{"message":"could not send webhook","status":500}`)
	testConfig.sparkpostTemplateId = mockSpTemplate

	fmt.Println("+++ sparkpost email error handling verified")

	// success
	assert.Equal(t, postTest("segment_webhook_valid", inst, ctx), `{"message":"success","status":200}`)
}

func TestGetOptimalSendTime(t *testing.T) {
	hourly := map[string]interface{}{
		"0": float64(583),
		"1": float64(414),
		"14": float64(1),
		"17": float64(721),
		"18": float64(1140),
		"23": float64(1138),
	}

	sendTime := getOptimalSendTime(hourly)
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
