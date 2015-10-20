# Package oauth1a
## Summary
An implementation of OAuth 1.0a in Go1.

## Installing
Run:

    go get github.com/kurrik/oauth1a

Include in your source:

    import "github.com/kurrik/oauth1a"

## Testing
Clone this repository, then run:

    go test -short

in the `oauth1a` directory.  To run an integration test, create a file named
CREDENTIALS in the library directory.  There should be four lines in this file,
in the following format:

    <Twitter consumer key>
    <Twitter consumer secret>
    <Twitter access token>
    <Twitter access token secret>

Then run:

    go test

This will run an integration test against the Twitter
`/account/verify_credentials.json` endpoint.

## Using
A good approach wil be to check `oauth1a_test.go` for usage.

As a vague example, here is code to configure the library for accessing Twitter:

    service := &oauth1a.Service{
    	RequestURL:   "https://api.twitter.com/oauth/request_token",
    	AuthorizeURL: "https://api.twitter.com/oauth/request_token",
    	AccessURL:    "https://api.twitter.com/oauth/request_token",
    	ClientConfig: &oauth1a.ClientConfig{
    		ConsumerKey:    "<your Twitter consumer key>",
    		ConsumerSecret: "<your Twitter consumer secret>",
    		CallbackURL:    "<your Twitter callback URL>",
    	},
    	Signer: new(oauth1a.HmacSha1Signer),
    }

To obtain user credentials:

    httpClient := new(http.Client)
    userConfig := &oauth1a.UserConfig{}
    userConfig.GetRequestToken(service, httpClient)
    url, _ := userConfig.GetAuthorizeURL(service)
    var token string
    var verifier string
    // Redirect the user to <url> and parse out token and verifier from the response.
    userConfig.GetAccessToken(token, verifier, service, httpClient)

Or if you have existing credentials:

    token := "<your access token>"
    secret := "<your access token secret>"
    userConfig := NewAuthorizedConfig(token, secret)

To send an authenticated request:

    httpRequest, _ := http.NewRequest("GET", "https://api.twitter.com/1/account/verify_credentials.json", nil)
    service.Sign(httpRequest, userConfig)
    var httpResponse *http.Response
    var err error
    httpResponse, err = httpClient.Do(httpRequest)


## Examples

[github.com/twittergo-examples/sign_in/main.go](https://github.com/kurrik/twittergo-examples/blob/master/sign_in/main.go) - A three legged example which uses Twitter's API.
To run, cd to the examples directory and then run:

    go run main.go -key=<TWITTER_CONSUMER_KEY> -secret=<TWITTER_CONSUMER_SECRET>

This will host a server on `localhost:10000` (use the `-port` flag to change the
port this runs on).  Navigate to `http://localhost:10000` and then follow the
sign in flow.

Note that this example implements a rudimentary session mechanism so that the
callback can be matched to the user who initiated the sign in session.  Otherwise,
it would be possible for one user to initiate a sign in session and another user
to complete it.  This is a best practice but imposes a requirement for the
auth flow to be stateful.  If you understand the risks in removing this check
from your application, it is possible to implement the flow in a stateless
manner.
