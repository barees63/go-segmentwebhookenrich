package main

var (
	// Lytics API Key 
	// Found in the "Manage Accounts" page in Lytics
	lyticsAPIKey = "LYTICS API KEY"

	// SparkPost API Key
	// Can generate a new SparkPost key under Account > API Keys in Sparkpost
	sparkpostAPIKey = "SPARKPOST API KEY"

	// Filter for which content documents to recommend
	// Can use '*' as wildcard. With multiple filters dictated
	// by AND/OR logic
	//
	// URL Filter:
	// -------------------------------------------
	// FILTER AND (url LIKE "www.example.com/blog/*")
	// -------------------------------------------
	// This will return any url on your site which contains `www.example.com/blog/` including:
	// `www.example.com/blog/`, `www.example.com/blog/post-name`, or `www.example.com/tagged/tag-name`
	// this filter should not include the 'http://' or 'https://' protocol. Select this
	// filter carefully so as not to recommend any page without choice content.
	//
	// Topic Filter:
	// -------------------------------------------
	// FILTER AND (global.Marketing > 0)
	// -------------------------------------------
	// Ensures that all documents recommended have the topic "Marketing" with a relevance
	// score above 0
	recommendationFilter = ""

	// Id of email template to send in SparkPost
	sparkpostTemplateId = "TEMPLATE ID"

	// If the user has hourly activity data in Lytics, and this flag is set to true,
	// we will look at the user's past activity and wait and send the email at the most
	// active time of day for the user. If no data exists, the email will send immediately
	// If this flag is set to false, emails will always send immediately
	sendAtOptimalHour = true
)
