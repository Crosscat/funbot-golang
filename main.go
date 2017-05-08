package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/magiconair/properties"
	_ "github.com/mattn/go-sqlite3"
)

// Variables used for command line parameters
var (
	Token              string
	BotID              string
	messageCounter     int
	messageWait        int
	ExcludedChannelIDs map[string]int

	DB  *sql.DB
	err error
)

func init() {

	props := properties.MustLoadFile("config.properties", properties.UTF8)
	flag.StringVar(&Token, "botid", props.MustGetString("BotID"), "Bot Token")

	var excludedChannels string
	flag.StringVar(&excludedChannels, "excluded", props.GetString("ExcludedChannels", ""), "Excluded Channels")
	flag.Parse()

	ExcludedChannelIDs = make(map[string]int)
	splitChannels := strings.Split(excludedChannels, ",")
	for _, element := range splitChannels {
		ExcludedChannelIDs[element] = 1
	}
}

func main() {

	messageWait = 50

	// Open DB connection
	DB, err = sql.Open("sqlite3", "./main.db")
	if err != nil {
		log.Fatal(err)
	}
	defer DB.Close()

	//	message := generateMessage("")
	//	message = scrubMessage(message)
	//	if message == "" {
	//		message = "i don't know any of these words"
	//	}
	//	fmt.Println(fmt.Sprintf("Message: %s", message))

	//	return

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

	if strings.HasPrefix(m.Content, "!talk") {
		message := generateMessage(m.Content[5:len(m.Content)])
		message = scrubMessage(message)
		if message == "" {
			message = "i don't know any of these words"
		}
		s.ChannelMessageSend(m.ChannelID, message)
		fmt.Println(fmt.Sprintf("Message: %s", message))

	} else {
		if messageCounter <= 0 && rand.Float64() < .05 {
			message := generateMessage(m.Content)
			if message != "" {
				message = scrubMessage(message)
				s.ChannelMessageSend(m.ChannelID, message)
				messageCounter = messageWait
			}
		}

		_, excluded := ExcludedChannelIDs[m.ChannelID]
		if !excluded {
			addMessageToDB(m.Content)
			messageCounter--
		}
	}
}

func scrubMessage(message string) string {
	r, _ := regexp.Compile("<@[\\S]*>")
	return r.ReplaceAllString(message, "")
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
	query := "select * from Words where Word In (?" + strings.Repeat(",?", len(messageArray)-1) + ") and Frequency > 0 order by Frequency asc"

	var params []interface{}
	for _, word := range messageArray {
		params = append(params, word)
	}

	row := DB.QueryRow(query, params...)

	return getWordContextFromRow(row)
}

func getWordArrayFromMessage(message string) []string {
	//remove punctuation?
	message = strings.ToLower(message)
	messageArray := strings.Split(removeWhiteSpace(message), " ")
	return messageArray
}

func getRandomWord() string {
	query := "Select Word From Words Order By Random()"

	var word string

	row := DB.QueryRow(query)
	err := row.Scan(&word)
	if err != nil {
		return ""
	}

	return word
}

func generateMessage(inquiry string) string {
	inquiry = strings.Trim(inquiry, " ")
	if inquiry == "" {
		inquiry = getRandomWord()
	}

	wordArray := getWordArrayFromMessage(inquiry)
	subjectWordContext := getSubjectWordContext(wordArray)
	if subjectWordContext.ID == 0 {
		return ""
	}

	associatedWordContext, comesAfter := getAssociatedWordFromPhrase(subjectWordContext, inquiry)

	if comesAfter {
		return generateStartPhrase(subjectWordContext, associatedWordContext) + generateEndPhrase(subjectWordContext, associatedWordContext)
	} else {
		return generateStartPhrase(associatedWordContext, subjectWordContext) + generateEndPhrase(associatedWordContext, subjectWordContext)
	}
}

func generatePhrase(surroundingType string, startWord []WordContext, phrase string) string {

	//stop condition:
	var stopFrequency int
	var frequency int
	var testWord WordContext

	for _, element := range startWord {
		testWord = element
		break
	}

	if surroundingType == "FollowingWordID" { //check for start frequency
		stopFrequency = testWord.StartFrequency
		frequency = testWord.Frequency
	} else {
		stopFrequency = testWord.EndFrequency
		frequency = testWord.Frequency
	}

	randVal := rand.Float64()
	if float64(stopFrequency)/float64(frequency) > randVal {
		fmt.Printf("Word: %s \n RandVal: %f \n StopFreq: %d \n Freq: %d\n", testWord.Word, randVal, stopFrequency, frequency)
		//fmt.Printf("Phrase: %s\n", phrase)
		//fmt.Println(startWord)
		return phrase
	}

	var nextID int
	//get next word id
	if len(startWord) == 2 {
		nextID = getIDWithSurroundingIDs(surroundingType, startWord[0].ID, startWord[1].ID)
	} else {
		nextID = getIDWithSurroundingIDs(surroundingType, startWord[0].ID)
	}

	if nextID == 0 {
		fmt.Println("Next id is 0")
		return phrase
	}

	//get word info
	nextWordContext := getWordInfoFromID(nextID)
	//fmt.Println(fmt.Sprintf("WordID: %d Word: %s", nextWordContext.ID, nextWordContext.Word))

	//add word to phrase
	phrase = phrase + " " + nextWordContext.Word

	//chain together 2 words
	nextWords := []WordContext{nextWordContext, startWord[0]}

	//recurse
	return generatePhrase(surroundingType, nextWords, phrase)
}

//picks word from phrase that directly follows or preceeds subject word
//returns associated wordcontext and true if word comes after subject word (false otherwise)
func getAssociatedWordFromPhrase(word WordContext, phrase string) (WordContext, bool) {
	wordArray := getWordArrayFromMessage(phrase)
	var query string
	var inWords []int

	var otherWords []interface{}
	if len(wordArray) > 1 {

		for _, element := range wordArray {
			if element == word.Word {
				continue
			}
			id := getWordInfo(element).ID
			otherWords = append(otherWords, id)
			inWords = append(inWords, id)
		}

		inString := "(?" + strings.Repeat(",?", len(otherWords)-1) + ")"
		query = fmt.Sprintf("Select WordID, TrailingWordID1, FollowingWordID1 from IDs where WordID=? And (TrailingWordID1 In %s Or FollowingWordID1 In %s) ORDER BY RANDOM()", inString, inString)
	} else {
		query = "Select WordID, TrailingWordID1, FollowingWordID1 from IDs where WordID=? ORDER BY RANDOM()"
	}

	//Creating new array containing all query values
	var params []interface{}
	params = append(params, word.ID)
	params = append(params, otherWords...)
	params = append(params, otherWords...)

	var id int
	var prevID int
	var nextID int
	row := DB.QueryRow(query, params...)
	err := row.Scan(&id, &prevID, &nextID)
	if err != nil {
		query = "Select WordID, TrailingWordID1, FollowingWordID1 from IDs where WordID=? ORDER BY RANDOM()"
		row := DB.QueryRow(query, params[0])
		err := row.Scan(&id, &prevID, &nextID)
		if err != nil {
			return word, false
		}
	}

	if prevID != 0 && contains(inWords, prevID) {
		return getWordInfoFromID(prevID), false
	} else {
		return getWordInfoFromID(nextID), true
	}
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func generateStartPhrase(startWord WordContext, associatedWordContext WordContext) string {
	wordContexts := []WordContext{startWord, associatedWordContext}
	if startWord == associatedWordContext {
		wordContexts = []WordContext{startWord}
	}

	phrase := reverseSentence(generatePhrase("FollowingWordID", wordContexts, ""))
	return phrase
}

func generateEndPhrase(startWord WordContext, associatedWordContext WordContext) string {
	wordContexts := []WordContext{associatedWordContext, startWord}
	if startWord == associatedWordContext {
		wordContexts = []WordContext{startWord}
	}

	phrase := generatePhrase("TrailingWordID", wordContexts, wordContexts[1].Word+" "+wordContexts[0].Word)
	return phrase
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
	var params []interface{}
	prefix := ""

	for index, id := range ids {
		buffer.WriteString(prefix)
		buffer.WriteString(fmt.Sprintf("%s%d=?", surroundingType, index+1))
		prefix = " And "

		params = append(params, id)
	}
	query := fmt.Sprintf("Select WordID from IDs where %s ORDER BY RANDOM()", buffer.String())

	row := DB.QueryRow(query, params...)
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
