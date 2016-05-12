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

// Expected format of incoming webhook from Segment
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
	router.Post("/post", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := appengine.NewContext(r)
		config.setClient(urlfetch.Client(ctx))
		config.enrichWebhook(w, r, ctx)
	}))

	http.HandleFunc("/", router.ServeHTTP)
}

// enrichWebhook accepts an incoming segment webhook event using
// the Lytics + Segment integration format. It looks up content
// recommendations for the user in the event, and sends this data
// to a webhook. In this example, it sends a formatted webhook to
// sparkpost to deploy which will email the with content suggested for them
func (c *Config) enrichWebhook(w http.ResponseWriter, r *http.Request, ctx appengine.Context) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", r.Method)

	// We expect the body of the post request to be a segment
	// track event containing lytics user data
	evt := &SegmentEvent{}
	if err := json.NewDecoder(r.Body).Decode(evt); err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "unrecognized webhook body"))
		return
	}

	// Check if event matches the expectation
	if c.event != nil {
		// Check if event name matches
		if c.event.name != "" && evt.EventName != c.event.name {
			w.WriteHeader(204)
			fmt.Fprintf(w, buildResponse(204, "not processed: event name did not match"))
			return
		}

		// Check if segment name matches
		friendlyName, ok := evt.Properties["_audience_friendly"].(string)
		if c.event.segment != "" && ok && friendlyName != c.event.segment {
			w.WriteHeader(204)
			fmt.Fprintf(w, buildResponse(204, "not processed: segment name did not match"))
			return
		}
	}

	// Email should exist
	if _, ok := evt.Properties["email"]; !ok {
		w.WriteHeader(400)
		fmt.Fprintf(w, buildResponse(400, "user does not have email"))
		return
	}

	ly := lytics.NewLytics(c.lyticsAPIKey, nil, c.client)

	// Get recommended content for the user
	recs, err := ly.GetUserContentRecommendation("emails", evt.Properties["email"].(string), c.recommendationFilter, 3, false)
	if err != nil || len(recs) == 0 {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "could not get recommendation for this user"))
		return
	}

	// Format your webhook however you like with our data this
	// example formulates a webhook which we send to sparkpost to
	// deploy an email to this user
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
			"template_id": c.sparkpostTemplateId,
		},
	}

	// Calculate the optimal time of day to send an email to this user
	if c.getOptimalHour {
		if sendTime := evt.SendTime(); sendTime != nil {
			payload["options"] = map[string]interface{}{
				"start_time": sendTime.Format(time.RFC3339),
			}
		}
	}

	// Send the payload data as a webhook
	reqBody, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "invalid outgoing webhook body"))
		return
	}

	req, err := http.NewRequest("POST", c.webhookUrl, bytes.NewReader(reqBody))
	req.Header.Set("Authorization", c.sparkpostAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	defer resp.Body.Close()
	if err != nil || resp.StatusCode != 200 {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "could not send webhook"))
		return
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, buildResponse(200, "success"))
}

// SendTime will look through the hourly data for the user
// and find the highest activity hour of the day, it returns the
// next time it will be the optimal hour for the user
func (e *SegmentEvent) SendTime() *time.Time {
	var (
		max         int
		optimalHour int
	)

	hourly, ok := e.Properties["hourly"].(map[string]interface{})

	if !ok {
		return nil
	}

	for key, val := range hourly {
		valInt := int(val.(float64))
		if valInt > max {
			max = valInt
			optimalHour, _ = strconv.Atoi(key)
		}
	}

	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), optimalHour, 0, 0, 0, time.UTC)

	if optimalHour == now.Hour() {
		// Send now
		return nil
	} else if date.Before(now) {
		// Send tomorrow at optimal hour
		date = date.AddDate(0, 0, 1)
	}

	return &date
}

func buildResponse(status int, msg string) string {
	output := map[string]interface{}{
		"status":  status,
		"message": msg,
	}

	resp, _ := json.Marshal(output)
	return string(resp)
}
