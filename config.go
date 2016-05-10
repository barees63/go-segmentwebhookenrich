package main

var (
	// Lytics API Key 
	// Found in the "Manage Accounts" page in Lytics
	lyticsAPIKey = "LYTICS API KEY"

	// Filter for which content documents to recommend
	// Can use '*' as wildcard. With multiple filters dictated by AND/OR logic
	// See README.md for examples. Leave as empty string for no filter.
	recommendationFilter = ""

	// URL to send webhook with recommendation data
	webhookUrl = "https://api.sparkpost.com/api/v1/transmissions"

	// If the user has hourly activity data in Lytics, and this flag is set to true,
	// we will look at the user's past activity and wait and compute the next optimal activity
	// hour for this user. This date may be included in the payload for the webhook
	getOptimalHour = true


	// OPTIONAL CONFIG PARAMS
	// (for our example, we need these to send an email with sparkpost)

	// Id of email template to send in SparkPost
	sparkpostTemplateId = "TEMPLATE ID"

	// SparkPost API Key
	// Can generate a new SparkPost key under Account > API Keys in Sparkpost
	sparkpostAPIKey = "SPARKPOST API KEY"
)
