package twitter

import (
	"strconv"
	"strings"

	"github.com/dghubble/go-twitter/twitter"
	"golang.org/x/oauth2"
)

var client *twitter.Client

func Initialize(bearerToken string) {
	config := &oauth2.Config{}
	token := &oauth2.Token{AccessToken: bearerToken}
	httpClient := config.Client(oauth2.NoContext, token)
	client = twitter.NewClient(httpClient)
}

func GetAllImagesFromUrl(url string) []string {
	split := strings.Split(url, "/")
	stringId := split[len(split)-1]
	id, _ := strconv.ParseInt(stringId, 0, 64)
	return GetAllImages(id)
}

func GetAllImages(id int64) []string {
	returnData := []string{}

	params := &twitter.StatusShowParams{}
	params.TweetMode = "extended"
	tweet, _, err := client.Statuses.Show(id, params)

	if err != nil {
		return returnData
	}

	if tweet.ExtendedEntities != nil && tweet.ExtendedEntities.Media != nil && len(tweet.ExtendedEntities.Media) > 1 {
		for i := 1; i < len(tweet.ExtendedEntities.Media); i++ {
			returnData = append(returnData, tweet.ExtendedEntities.Media[i].MediaURL)
		}
	}

	return returnData
}
