package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/magiconair/properties"
	_ "github.com/mattn/go-sqlite3"
)

// Variables used for command line parameters
var (
	Token string
	BotID string

	DB  *sql.DB
	err error
)

func init() {

	props := properties.MustLoadFile("config.properties", properties.UTF8)
	flag.StringVar(&Token, "t", props.MustGetString("BotID"), "Bot Token")
	flag.Parse()
}

func main() {

	// Open DB connection
	DB, err = sql.Open("sqlite3", "./main.db")
	if err != nil {
		log.Fatal(err)
	}
	defer DB.Close()

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
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
	BotID = u.ID

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
	return
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == BotID {
		return
	}

	if strings.HasPrefix(m.Content, "!talk ") {
		s.ChannelMessageSend(m.ChannelID, generateMessage(m.Content[6:len(m.Content)]))
	} else {
		addMessageToDB(m.Content)
	}
}

//gets word information from db, returns it as a map.  If word does not exist, returns empty map.
func getWordInfo(word string) WordContext {
	stmt, err := DB.Prepare("select * from Words where Word = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	row := stmt.QueryRow(word)

	return getWordContextFromRow(row)
}

//gets word information from db, returns it as a map.  If word does not exist, returns empty map.
func getWordInfoFromID(id int) WordContext {
	stmt, err := DB.Prepare("select * from Words where ID = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	row := stmt.QueryRow(id)

	return getWordContextFromRow(row)
}

func getWordContextFromRow(row *sql.Row) WordContext {
	var id int
	var word string
	var frequency int
	var startFrequency int
	var endFrequency int
	err = row.Scan(&word, &id, &frequency, &endFrequency, &startFrequency)
	if err != nil {
		fmt.Println(err)
	}

	info := WordContext{Word: word, ID: id, Frequency: frequency, StartFrequency: startFrequency, EndFrequency: endFrequency}

	return info
}

//updates word frequency in Words table.  Inserts new record if word does not exist.
func updateWord(word string, start bool, end bool, data WordContext) int {
	id := data.ID
	if id == 0 {
		id = insertWord(word)
	}

	stmt, err := DB.Prepare("Update Words Set Frequency = ?, EndFrequency = ?, StartFrequency = ? Where ID = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	if start {
		data.StartFrequency++
	}
	if end {
		data.EndFrequency++
	}
	data.Frequency++

	_, err = stmt.Exec(data.Frequency, data.EndFrequency, data.StartFrequency, id)
	if err != nil {
		log.Fatal(err)
	}
	return id
}

func addMessageToDB(message string) {
	messageArray := getWordArrayFromMessage(message)

	idArray := make([]int, len(messageArray))
	for index, word := range messageArray { //for each word in message
		data := getWordInfo(word) //check if word exists in table

		id := updateWord(word, index == 0, index == len(messageArray)-1, data)
		idArray[index] = id //add id to id array
	}

	insertMap := make(map[string]int) //create map for storing ids for insertion
	for index := range idArray {      //for each id in message
		for i := -3; i <= 3; i++ { //check preceding 3 ids and following 3 ids
			offset := index + i

			var value int
			if offset >= 0 && offset < len(idArray) { //verify id is within array bounds
				value = idArray[offset] //set value for insert map
			} else {
				value = 0
			}

			var key string
			switch i { //set key for insert map
			case -3:
				key = "TrailingWordID3"
			case -2:
				key = "TrailingWordID2"
			case -1:
				key = "TrailingWordID1"
			case 0:
				key = "WordID"
			case 1:
				key = "FollowingWordID1"
			case 2:
				key = "FollowingWordID2"
			case 3:
				key = "FollowingWordID3"
			}

			insertMap[key] = value
		}

		insertIDs(insertMap) //insert values into IDs table
	}
}

//inserts word into Word table, returns ID
func insertWord(word string) int {
	stmt, err := DB.Prepare("insert into Words (Word, Frequency, EndFrequency, StartFrequency) values(?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(word, 0, 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	return getWordInfo(word).ID
}

//inserts ids into IDs table
func insertIDs(insertMap map[string]int) {
	stmt, err := DB.Prepare("insert into IDs (WordID, FollowingWordID1, FollowingWordID2, FollowingWordID3, TrailingWordID1, TrailingWordID2, TrailingWordID3) values(?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(insertMap["WordID"], insertMap["FollowingWordID1"], insertMap["FollowingWordID2"], insertMap["FollowingWordID3"], insertMap["TrailingWordID1"], insertMap["TrailingWordID2"], insertMap["TrailingWordID3"])
	if err != nil {
		log.Fatal(err)
	}
}

func removeWhiteSpace(str string) string {
	return strings.Join(strings.Fields(str), " ")
}

func getSubjectWordContext(messageArray []string) WordContext {
	var buffer bytes.Buffer
	prefix := ""
	for _, word := range messageArray {
		buffer.WriteString(prefix)
		buffer.WriteString("'" + word + "'")
		prefix = ","
	}

	row := DB.QueryRow("select * from Words where Word In (" + buffer.String() + ") and Frequency > 1 order by Frequency asc")

	return getWordContextFromRow(row)
}

func getWordArrayFromMessage(message string) []string {
	//remove punctuation?
	message = strings.ToLower(message)
	messageArray := strings.Split(removeWhiteSpace(message), " ")
	return messageArray
}

func generateMessage(inquiry string) string {
	wordArray := getWordArrayFromMessage(inquiry)
	subjectWordContext := getSubjectWordContext(wordArray)

	return generateStartPhrase(subjectWordContext) + generateEndPhrase(subjectWordContext)
}

func generatePhrase(surroundingType string, startWord WordContext, phrase string) string {

	//stop condition:
	var stopFrequency int
	if surroundingType == "FollowingWordID" { //check for start frequency
		stopFrequency = startWord.StartFrequency
	} else {
		stopFrequency = startWord.EndFrequency
	}

	randVal := rand.Float64()
	if float64(stopFrequency)/float64(startWord.Frequency) > randVal {
		//fmt.Printf("Word: %s \n RandVal: %f \n StopFreq: %d \n Freq: %d", startWord.Word, randVal, stopFrequency, startWord.Frequency)
		return phrase
	}

	//get next word id
	nextID := getIDWithSurroundingIDs(surroundingType, startWord.ID)
	if nextID == 0 {
		return phrase
	}

	//get word info
	nextWordContext := getWordInfoFromID(nextID)

	//add word to phrase
	phrase = phrase + " " + nextWordContext.Word

	//recurse
	return generatePhrase(surroundingType, nextWordContext, phrase)
}

func generateStartPhrase(startWord WordContext) string {
	return reverseSentence(generatePhrase("FollowingWordID", startWord, startWord.Word))
}

func generateEndPhrase(startWord WordContext) string {
	return generatePhrase("TrailingWordID", startWord, "")
}

func reverseSentence(sentence string) string {
	words := strings.Split(sentence, " ")
	var buffer bytes.Buffer
	prefix := ""
	for i := len(words) - 1; i >= 0; i-- {
		buffer.WriteString(prefix)
		buffer.WriteString(words[i])
		prefix = " "
	}
	return buffer.String()
}

func getIDWithSurroundingIDs(surroundingType string, ids ...int) int {
	var buffer bytes.Buffer
	prefix := ""
	for index, id := range ids {
		buffer.WriteString(prefix)
		buffer.WriteString(fmt.Sprintf("%s%d='%d'", surroundingType, index+1, id))
		prefix = " And "
	}
	query := fmt.Sprintf("Select WordID from IDs where %s ORDER BY RANDOM()", buffer.String())
	//fmt.Println(query)
	row := DB.QueryRow(query)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return 0
	}
	return id
}

type WordContext struct {
	Word           string
	ID             int
	Frequency      int
	StartFrequency int
	EndFrequency   int
}

type IDContext struct {
	WordID          int
	FollowingWordID []int
	TrailingWordID  []int
}
