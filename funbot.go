package main

import (
	"fmt"
	"funbot/ai"
	"funbot/codenames"
	"funbot/db"
	"funbot/reddit"
	"funbot/twitter"
	"funbot/utils"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	//"github.com/robfig/cron"
)

// Variables used for command line parameters
var (
	botConfig      Config
	botID          string
	messageCounter int
	messageWait    int
	messageChance  float64

	adminID string

	database db.BotDB
	err      error

	botAI ai.FunAI
)

func init() {
	//Load config data from properties file
	configFilePath := os.Args[1]
	if string(configFilePath[1]) != ":" {
		configFilePath = "./" + configFilePath
	}
	botConfig = InitializeConfig(configFilePath)
}

func main() {

	twitter.Initialize(botConfig.TwitterToken)
	messageWait = botConfig.MessageWait
	messageChance = botConfig.MessageChance
	adminID = botConfig.AdminId

	database.Open("./main.db")
	fmt.Println("Opened DB")
	defer database.Close()

	botAI.BotDB = &database

	// Setup reddit functionality
	if botConfig.RedditSettings != "" {
		reddit.Initialize(botAI.BotDB, botConfig.RedditSettings)
	}

	// Setup codenames functionality
	if botConfig.CodeNameWordListFile != "" {
		codenames.Setup(botConfig.CodeNameWordListFile, botConfig.AdminId)
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + botConfig.Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Get the account information.
	u, err := dg.User("@me")
	if err != nil {
		fmt.Println("error obtaining account details,", err)
	}

	// Store the account ID for later use.
	botID = u.ID

	// Register messageCreate as a callback for the messageCreate events.
	dg.AddHandler(messageCreate)

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	// Simple way to keep program running until CTRL-C is pressed.
	<-make(chan struct{})

	fmt.Println("Closed DB")
	return
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == botID {
		return
	}

	if m.Author.ID == adminID {
		if strings.HasPrefix(m.Content, "!purge") {
			number, _ := strconv.Atoi(strings.TrimSpace(m.Content[6:len(m.Content)]))
			messages, _ := s.ChannelMessages(m.ChannelID, number, "", "", "")
			fmt.Println(messages[0].Content)
			for _, ele := range messages {
				fmt.Printf("Purging message id %s", ele.ID)
				s.ChannelMessageDelete(m.ChannelID, ele.ID)
			}
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Deleted last %d messages", number))
			return
		}
	}

	if strings.HasPrefix(m.Content, "!talk") {
		message := botAI.GenerateMessage(m.Content[5:len(m.Content)])
		message = scrubMessage(message)
		if message == "" {
			message = "i don't know any of these words"
		}
		s.ChannelMessageSend(m.ChannelID, message)
		fmt.Println(fmt.Sprintf("Message: %s", message))

	} else if strings.HasPrefix(m.Content, "!reddit") { //reddit
		sub := strings.TrimSpace(m.Content[7:len(m.Content)])

		includeNSFW := utils.ArrayContains(botConfig.NSFW, m.ChannelID)
		message := reddit.GetRandomTopSubPost(sub, includeNSFW)
		s.ChannelMessageSend(m.ChannelID, message)

	} else if strings.HasPrefix(m.Content, "!addcommand") {
		splitMessage := strings.SplitN(strings.TrimSpace(m.Content[11:len(m.Content)]), " ", 2)
		if !strings.HasPrefix(splitMessage[0], "!") {
			return
		}
		botAI.AddCommand(splitMessage[0], splitMessage[1])
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Added command %s", splitMessage[0]))

	} else if strings.HasPrefix(m.Content, "!cn") {
		message := codenames.ProcessCommand(m, s)
		s.ChannelMessageSend(m.ChannelID, message)

	} else if strings.HasPrefix(m.Content, "!twit") {
		urls := twitter.GetAllImagesFromUrl(m.Content[5:len(m.Content)])
		for i := 0; i < len(urls); i++ {
			s.ChannelMessageSend(m.ChannelID, urls[i])
		}

	} else if strings.HasPrefix(m.Content, "!") {
		mbMessage := botAI.GetResponseFromCommand(m.Content)
		if mbMessage != "" {
			s.ChannelMessageSend(m.ChannelID, mbMessage)
			return
		}

	} else {
		if messageCounter <= 0 && rand.Float64() < messageChance {
			message := botAI.GenerateMessage(m.Content)
			if message != "" {
				message = scrubMessage(message)
				s.ChannelMessageSend(m.ChannelID, message)
				messageCounter = messageWait
			}
		}
		_, excluded := botConfig.ExcludedChannelIDs[m.ChannelID]
		if !excluded {
			botAI.AddMessageToDB(m.Content)
			messageCounter--
		}
	}
}

func sendMessageToChannel(s *discordgo.Session, channelID string, message string) {
	s.ChannelMessageSend(channelID, message)
}

func scrubMessage(message string) string {
	r, _ := regexp.Compile("<@[\\S]*>")
	s := r.ReplaceAllString(message, "")
	s = strings.Replace(s, "?", "", -1)
	s = strings.Replace(s, "\"", "", -1)
	s = strings.Replace(s, ",", "", -1)
	return s
}
