package main

import (
	"fmt"
	"funbot/ai"
	"funbot/db"
	"math/rand"
	"os"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Variables used for command line parameters
var (
	botConfig      Config
	botID          string
	messageCounter int
	messageWait    int

	database db.BotDB
	err      error

	botAI ai.FunAI
)

func init() {
	//Load config data from properties file
	configFilePath := os.Args[1]
	botConfig = InitializeConfig("./" + configFilePath)
}

func main() {

	messageWait = 50

	database.Open("./main.db")
	fmt.Println("Opened DB")

	botAI.BotDB = &database

	//	message := generateMessage("")
	//	message = scrubMessage(message)
	//	if message == "" {
	//		message = "i don't know any of these words"
	//	}
	//	fmt.Println(fmt.Sprintf("Message: %s", message))

	//return

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
	database.Close()
	return
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == botID {
		return
	}

	if strings.HasPrefix(m.Content, "!talk") {
		message := botAI.GenerateMessage(m.Content[5:len(m.Content)])
		message = scrubMessage(message)
		if message == "" {
			message = "i don't know any of these words"
		}
		s.ChannelMessageSend(m.ChannelID, message)
		fmt.Println(fmt.Sprintf("Message: %s", message))

	} else {
		if messageCounter <= 0 && rand.Float64() < .05 {
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

func scrubMessage(message string) string {
	r, _ := regexp.Compile("<@[\\S]*>")
	return r.ReplaceAllString(message, "")
}
