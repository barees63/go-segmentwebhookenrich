package main

type Config struct {
	// required
	lyticsAPIKey         string
	webhookUrl           string
	getOptimalHour       bool

	// optional
	recommendationFilter string
	event                *event
	sparkpostTemplateId  string
	sparkpostAPIKey      string
}

type event struct {
	name string
	segment string
}

var (
	config = Config {
		// Lytics API Key 
		// Found in the "Manage Accounts" page in Lytics
		lyticsAPIKey: "LYTICS API KEY",

		// Filter for which content documents to recommend
		// Can use '*' as wildcard. With multiple filters dictated by AND/OR logic
		// See README.md for examples. Leave as empty string for no filter.
		recommendationFilter: "",

		// URL to send webhook with recommendation data
		webhookUrl: "https://api.sparkpost.com/api/v1/transmissions",

		// If the user has hourly activity data in Lytics, and this flag is set to true,
		// we will look at the user's past activity and wait and compute the next optimal activity
		// hour for this user. This date may be included in the payload for the webhook
		getOptimalHour: true,


		// Filtering for which events to process.
		// If you want to accept every event that comes through, do not set this field.
		event: &event{
			// Name of event (segment_entered or segment_exited)
			name: "segment_exited",

			// Name of segment should match API name of segment in Lytics
			segment: "sample_segment_name",
		},

		// Id of email template to send in SparkPost (optional for this example)
		sparkpostTemplateId: "SPARKPOST TEMPLATE ID",

		// SparkPost API Key (optional for this example)
		// Can generate a new SparkPost key under Account > API Keys in Sparkpost
		sparkpostAPIKey: "SPARKPOST API",
	}
)