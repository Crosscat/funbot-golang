package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/magiconair/properties"
)

type Config struct {
	Token                string
	ExcludedChannelIDs   map[string]int
	MessageWait          int
	MessageChance        float64
	Subreddits           map[string][]string
	NSFW                 []string
	AdminId              string
	RedditSettings       string
	CodeNameWordListFile string
	TwitterToken         string
}

//Loads config properties from properties file and returns Config object
func InitializeConfig(configFile string) Config {
	fmt.Println("Loading from " + configFile)
	props := properties.MustLoadFile(configFile, properties.UTF8)

	//Token
	var token string
	flag.StringVar(&token, "token", props.MustGetString("Token"), "Bot Token")

	//Excluded channels
	var excludedChannels string
	flag.StringVar(&excludedChannels, "excluded", props.GetString("ExcludedChannels", ""), "Excluded Channels")

	//Message Wait
	var messageWait int
	flag.IntVar(&messageWait, "messagewait", props.GetInt("MessageWait", 50), "Message Wait")

	//Message Chance
	var messageChance float64
	flag.Float64Var(&messageChance, "messageChance", props.GetFloat64("MessageChance", 0.05), "Message Chance")

	//Reddit
	var redditSettings string
	flag.StringVar(&redditSettings, "redditSettings", props.GetString("RedditSettings", ""), "Reddit Settings")

	//Subreddits
	var subreddits string
	flag.StringVar(&subreddits, "subreddits", props.GetString("Subreddits", ""), "Subreddits")

	//NSFW
	var nsfw string
	flag.StringVar(&nsfw, "nsfw", props.GetString("NSFW", ""), "NSFW")

	//Admin ID
	var adminID string
	flag.StringVar(&adminID, "adminId", props.GetString("AdminId", ""), "Admin Id")

	//CodeNameWordListFile
	var codeNameWordListFile string
	flag.StringVar(&codeNameWordListFile, "codeNameWordListFile", props.GetString("CodeNameWordListFile", ""), "CodeName Word List File")

	//TwitterToken
	var twitterToken string
	flag.StringVar(&twitterToken, "twitterToken", props.GetString("TwitterToken", ""), "Twitter Access Token")

	flag.Parse()

	//Convert excluded channel ids to map
	excludedChannelIDs := make(map[string]int)
	splitChannels := strings.Split(excludedChannels, ",")
	for _, element := range splitChannels {
		excludedChannelIDs[element] = 1
	}

	//Convert subreddits to map
	subredditMap := make(map[string][]string)
	if subreddits != "" {
		splitChannels = strings.Split(subreddits, ";")
		for _, element := range splitChannels {
			channel := strings.Split(element, "-")[0]
			subs := element[len(channel)+1 : len(element)]
			subsSplit := strings.Split(subs, ",")
			subredditMap[channel] = subsSplit
		}
	}

	//Convert nsfw to array
	nsfwSplit := SplitTrim(nsfw, ",")

	fmt.Println("Token: " + token)
	fmt.Println("Excluded Channel IDs: " + excludedChannels)
	fmt.Printf("MessageWait: %d\n", messageWait)
	fmt.Printf("MessageChance: %f\n", messageChance)
	fmt.Printf("NSFW channels: %s\n", nsfw)

	return Config{
		Token:                token,
		ExcludedChannelIDs:   excludedChannelIDs,
		MessageWait:          messageWait,
		MessageChance:        messageChance,
		RedditSettings:       redditSettings,
		Subreddits:           subredditMap,
		NSFW:                 nsfwSplit,
		AdminId:              adminID,
		CodeNameWordListFile: codeNameWordListFile,
		TwitterToken:         twitterToken,
	}
}

func SplitTrim(input string, delim string) []string {
	split := strings.Split(input, delim)
	for index, ele := range split {
		split[index] = strings.TrimSpace(ele)
	}
	return split
}
