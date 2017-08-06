package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/magiconair/properties"
)

type Config struct {
	Token              string
	ExcludedChannelIDs map[string]int
}

//Loads config properties from properties file and returns Config object
func InitializeConfig(configFile string) Config {
	fmt.Println("Loading from " + configFile)

	var token string
	props := properties.MustLoadFile(configFile, properties.UTF8)
	flag.StringVar(&token, "token", props.MustGetString("Token"), "Bot Token")

	var excludedChannels string
	flag.StringVar(&excludedChannels, "excluded", props.GetString("ExcludedChannels", ""), "Excluded Channels")
	flag.Parse()

	excludedChannelIDs := make(map[string]int)
	splitChannels := strings.Split(excludedChannels, ",")
	for _, element := range splitChannels {
		excludedChannelIDs[element] = 1
	}

	fmt.Println("Token: " + token)
	fmt.Println("Excluded Channel IDs: " + excludedChannels)

	return Config{Token: token, ExcludedChannelIDs: excludedChannelIDs}
}
