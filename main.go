package main

import (
	"appengine"
	"appengine/urlfetch"
	"encoding/json"
	"fmt"
	"github.com/bmizerany/pat"
	lytics "github.com/lytics/go-lytics"
	sp "github.com/SparkPost/gosparkpost"
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
	router.Post("/post", http.HandlerFunc(webhookToSparkpost))
	http.HandleFunc("/", router.ServeHTTP)
}

func webhookToSparkpost(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", r.Method)

	// expect the body of the post request to be a segment
	// track event containing lytics user data
	evt := &SegmentEvent{}
	if err := json.NewDecoder(r.Body).Decode(evt); err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "unrecognized webhook body", nil))
		return
	}

	// email should exist
	if _, ok := evt.Properties["email"]; !ok {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "user does not have email", nil))
		return
	}

	// get recommended content for the user
	ly := lytics.NewLytics(lyticsAPIKey, nil)
	ly.SetClient(client)

	if recommendationFilter != "" {
		recommendationFilter += " FROM content"
	}

	recs, err := ly.GetUserContentRecommendation("emails", evt.Properties["email"].(string), recommendationFilter, 1, false)
	if err != nil || len(recs) == 0 {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "could not get recommendation for this user", nil))
		return
	}

	recc := recs[0]

	// send data to sparkpost to format and send email
	spCfg := &sp.Config {
		BaseUrl: "https://api.sparkpost.com",
		ApiKey: sparkpostAPIKey,
		ApiVersion: 1,
	}

	spClient := sp.Client{}
	if err := spClient.Init(spCfg); err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, buildResponse(500, "invalid sparkpost credentials", nil))
		return
	}

	spClient.Client = client

	msg := &sp.Transmission {
		Recipients: []string{evt.Properties["email"].(string)},
		SubstitutionData: map[string]interface{}{
			"url": recc.Url,
			"title": recc.Title,
		},
		Content: map[string]string {
			"template_id": sparkpostTemplateId,
		},
	}

	hourly, ok := evt.Properties["hourly"].(map[string]interface{})

	if ok && sendAtOptimalHour {
		if sendTime := getOptimalSendTime(hourly); sendTime != nil {
			msg.Options = &sp.TxOptions {
				StartTime: sendTime.Format(time.RFC3339),
			}
		}
	}

	id, _, err := spClient.Send(msg)
	if err != nil {
		w.WriteHeader(500)
		ctx.Infof("%s", err)
		fmt.Fprintf(w, buildResponse(500, "failed to send to sparkpost", nil))
		return
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, buildResponse(200, "success", id))
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
	} else if date.After(now) {
		// send today at optimalHour today
		return &date
	}

	// send tomorrow at optimal hour
	date = date.AddDate(0, 0, 1)
	return &date
}

func buildResponse(status int, msg, id interface{}) string {
	output := map[string]interface{}{
		"status": status,
		"message": msg,
		"id": id,
	}

	resp, _ := json.Marshal(output)
	return string(resp)
}