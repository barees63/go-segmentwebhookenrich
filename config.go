package main

import "net/http"

type Config struct {
	// required
	lyticsAPIKey   string
	webhook        string
	webhooks       map[string]map[string]string
	getOptimalHour bool

	// optional
	recommendationFilter string
	event                *Event
	client               *http.Client
}

func (c *Config) setClient(client *http.Client) {
	c.client = client
}

type Event struct {
	name    string
	segment string
}

var (
	config = &Config{

		// Lytics API Key
		// Found in the "Manage Accounts" page in Lytics
		lyticsAPIKey: "LYTICS API KEY",

		// key of the webhook we'd like to use from the
		// webhooks map below
		webhook: "sparkpost",

		// Map of possible webhook endpoints to use in the
		webhooks: map[string]map[string]string{

			// This is the default webhook we use in the base code,
			// it sends a transmission to sparkpost, which will deploy an email.
			"sparkpost": map[string]string{

				// SparkPost API Key
				// Can generate a new SparkPost key under Account > API Keys in Sparkpost
				"apikey": "SPARKPOST API KEY",

				// URL to send webhook with recommendation data
				"url": "https://api.sparkpost.com/api/v1/transmissions",

				// Id of email template to send in SparkPost
				"template": "SPARKPOST TEMPLATE ID",
			},
		},

		// If the user has hourly activity data in Lytics, and this flag is set to true,
		// we will look at the user's past activity and wait and compute the next optimal activity
		// hour for this user. This date may be included in the payload for the webhook
		getOptimalHour: true,

		// Filter for which content documents to recommend
		// Can use '*' as wildcard. With multiple filters dictated by AND/OR logic
		// See README.md for examples. Leave as empty string for no filter.
		recommendationFilter: `FILTER AND (url LIKE "www.example.com/*") FROM content`,

		// Filtering for which events to process.
		// If you want to accept every event that comes through, do not set this field.
		event: &Event{

			// Name of event (segment_entered or segment_exited)
			name: "segment_entered",

			// Name of segment should match API name of segment in Lytics
			segment: "sample_segment_name",
		},
	}
)
