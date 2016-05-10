package main

import (
	"appengine"
	"appengine/urlfetch"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bmizerany/pat"
	lytics "github.com/lytics/go-lytics"
	"net/http"
	"strconv"
	"time"
)

type SegmentEvent struct {
	Version    int                    `json:"version,omitempty"`
	Type       string                 `json:"type,omitempty"`
	UserId     string                 `json:"userId,omitempty"`
	EventName  string                 `json:"event,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Timestamp  time.Time              `json:"timestamp,omitempty"`
}

func init() {
	router := pat.New()
	router.Post("/post", http.HandlerFunc(enrichWebhook))
	http.HandleFunc("/", router.ServeHTTP)
}

func enrichWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", r.Method)

	// we expect the body of the post request to be a segment
	// track event containing lytics user data
	evt := &SegmentEvent{}
	if err := json.NewDecoder(r.Body).Decode(evt); err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "unrecognized webhook body"))
		return
	}

	// check if event matches the expectation
	if config.event != nil {
		// check if event name matches
		if config.event.name != "" && evt.EventName != config.event.name {
			w.WriteHeader(204)
			fmt.Fprintf(w, buildResponse(204, "not processed: event name did not match"))
			return
		}

		// check if segment name matches
		friendlyName, ok := evt.Properties["_audience_friendly"].(string)
		if config.event.segment != "" && ok && friendlyName != config.event.segment {
			w.WriteHeader(204)
			fmt.Fprintf(w, buildResponse(204, "not processed: segment name did not match"))
			return
		}
	}

	// email should exist
	if _, ok := evt.Properties["email"]; !ok {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "user does not have email"))
		return
	}

	// get recommended content for the user
	ly := lytics.NewLytics(config.lyticsAPIKey, nil)
	ly.SetClient(client)

	if config.recommendationFilter != "" {
		config.recommendationFilter += " FROM content"
	}

	recs, err := ly.GetUserContentRecommendation("emails", evt.Properties["email"].(string), config.recommendationFilter, 1, false)
	if err != nil || len(recs) == 0 {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "could not get recommendation for this user"))
		return
	}

	// Format your webhook however you like with our data this
	// example formulates a webhook which will send an email using sparkpost
	payload := map[string]interface{}{
		"recipients": []map[string]interface{}{
			map[string]interface{}{
				"address": evt.Properties["email"].(string),
				"substitution_data": map[string]interface{}{
					"data": recs[0],
				},
			},
		},
		"content": map[string]string{
			"template_id": config.sparkpostTemplateId,
		},
	}

	hourly, ok := evt.Properties["hourly"].(map[string]interface{})

	if ok && config.getOptimalHour {
		if sendTime := getOptimalSendTime(hourly); sendTime != nil {
			payload["options"] = map[string]interface{}{
				"start_time": sendTime.Format(time.RFC3339),
			}
		}
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "invalid outgoing webhook body"))
		return
	}

	req, err := http.NewRequest("POST", config.webhookUrl, bytes.NewReader(reqBody))
	req.Header.Set("Authorization", config.sparkpostAPIKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil || res.StatusCode != 200 {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "could not send send webhook"))
		return
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, buildResponse(200, "success"))
}

func getOptimalSendTime(hourly map[string]interface{}) *time.Time {
	var (
		max int
		optimalHour int
	)

	for key, val := range hourly {
		valInt := int(val.(float64))
		if valInt > max {
			max = int(val.(float64))
			optimalHour, _ = strconv.Atoi(key)
		}
	}

	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), optimalHour, 0, 0, 0, time.UTC)

	if optimalHour == now.Hour() {
		// send now
		return nil
	} else if date.Before(now) {
		// send tomorrow at optimal hour
		date = date.AddDate(0, 0, 1)
	}

	return &date
}

func buildResponse(status int, msg string) string {
	output := map[string]interface{}{
		"status": status,
		"message": msg,
	}

	resp, _ := json.Marshal(output)
	return string(resp)
}