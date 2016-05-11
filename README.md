# go-segmentwebhookenrich

go-segmentwebhhokenrich is a Google App Engine app used to subscribe to a [Segment.com](https://segment.com/) event webhook coming from a [Lytics](http://www.getlytics.com/) segment enter/exited trigger and enrich contained profile with suggested content and optimal send time, to be used for email or any other kind of interaction. 

In the example code provided, we send a formatted webhook to [SparkPost](https://www.sparkpost.com/) which will deploy an email to the user at their optimal activity time including the suggested content. The base code of this app is flexible and can be edited to send the enriched data as a webhook to any url.

This app assumes you have a Lytics account, with at least one segment.com trigger export running to capture segment entered and exited events. To use this app, you should configure a webhooks integration on the segment.com source collecting these triggers. The webhook url should be the `[url of this app]/post`.


## Configuration

There are a few configuration variables which can be set in [`config.go`](https://github.com/lytics/go-segmentwebhookenrich/blob/master/config.go)

#### 1. lyticsAPIKey `string` (required)

This is required to make content recommendation requests. Your Lytics API Key can be found by going to `Account > Manage Acccounts` while logged in to Lytics. Use the Full API key.

#### 2. webhookUrl `string` (required)

The URL to POST the recommendation data to. For Sparkpost this will be to their transmissions API, but this URL could be anything. For testing try a service like [RequestBin](http://requestb.in/)

#### 3. getOptimalHour `bool` (required)

A flag to turn on or off the inclusion of the next optimal activity time in the payload of the outgoing webhook. If field is set to true we will look at the hourly data for the user and select the hour with the most activity in the past. Using this, we can get a timestamp to be included in the payload representing the next best time to interact with this user. 

If the current hour is the most optimal for the user, or they do not have any hourly activity data, this timestamp will not be included in the payload. In our example, we send this as the `start_time` to the sparkpost api, meaning the email will not be sent until the next optimal hour. Or immediately if this field is not set.

#### 4. event `*Event` (optional)

Once your webhook integration is configured your Segment source will send all incoming events to this app. By setting the `event` field `name` and `segment` we can select which events we actually want to process. If not set, the app will try to process all events. 

- **event.name** `string` - The name of the segment event. With lytics triggers this should be `segment_entered` or `segment_exited`.
- **event.segment** `string` - The slug of the segment in Lytics. (Make sure API Access is enabled for the segment).

#### 5. recommendationFilter `string` (optional)

This field can be used to filter document returned by the Lytics content recommendation API. This is an optional configuration variable, with no filter, the recommendation API may return any web document on your website for recommendation based on the users interests. However, with this filter you can select only documents of certain urls, meta attributes, associated topics, etc to return. This is combined with AND/OR logic. Consider these examples:

**URL Filter (Can use `*` as wildcard)**
>```
FILTER AND (url LIKE "www.example.com/blog/*")
```
This filter will include all documents matching the url pattern `www.example.com/blog/*` this could include `www.example.com/blog/post/1`, `www.example.com/blog/tagged/example`, etc. Be sure to choose a URL filter carefully, so as not to potentially recommend any content you wouldn't like to promote.


**Topic Filter**
>```
FILTER AND (global.Marketing > 0)
```
This filter will include all documents classified to have the topic `Marketing` with a relevence value greater than 0. You can view a list of all your topics for your content in the content section of your lytics account. All topics should be prefixed with `global.`


**Other Filters**
>```
FILTER AND (meta CONTAINS "og:type/article")
```
The filter above will only select documents with the og type article.


**Multiple Filters**
>```
FILTER OR (meta CONTAINS "og:type/article", global.developers > 0)
```
```
FILTER AND (url LIKE "www.example.com/products/*", global.mobile > 0)
```


#### 6. sparkpostTemplateId & sparkpostAPIKey `string` (optional)
These are used in the base code as an example, if you are not sending your webhook to sparkpost feel free to delete these fields and add anything you may need for your webhook.