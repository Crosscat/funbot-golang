package ai

import (
	"database/sql"
	"fmt"
	"funbot/db"
	"math/rand"
	"strings"

	"bytes"
)

type FunAI struct {
	BotDB *db.BotDB
}

func (ai *FunAI) GetResponseFromCommand(command string) string {
	var response string

	row := ai.BotDB.QueryRow("select Response from Commands where Command = ?", command)
	err := row.Scan(&response)
	if err != nil {
		return ""
	}
	return response
}

func (ai *FunAI) AddCommand(command string, response string) {
	query := "insert into Commands (Command, Response) values(?, ?)"
	ai.BotDB.Update(query, command, response)
}

//gets word information from db, returns it as a map.  If word does not exist, returns empty map.
func (ai *FunAI) getWordInfo(word string) wordContext {
	query := "select * from Words where Word = ?"
	row := ai.BotDB.QueryRow(query, word)
	return getWordContextFromRow(row)
}

//gets word information from db, returns it as a map.  If word does not exist, returns empty map.
func (ai *FunAI) getWordInfoFromID(id int) wordContext {
	query := "select * from Words where ID = ?"
	row := ai.BotDB.QueryRow(query, id)
	return getWordContextFromRow(row)
}

func getWordContextFromRow(row *sql.Row) wordContext {
	var id int
	var word string
	var frequency int
	var startFrequency int
	var endFrequency int
	err := row.Scan(&word, &id, &frequency, &endFrequency, &startFrequency)
	if err != nil {
		fmt.Println(err)
	}

	info := wordContext{word: word, id: id, frequency: frequency, startFrequency: startFrequency, endFrequency: endFrequency}

	return info
}

//updates word frequency in Words table.  Inserts new record if word does not exist.
func (ai *FunAI) updateWord(word string, start bool, end bool, data wordContext) int {
	id := data.id
	if id == 0 {
		id = ai.insertWord(word)
	}

	if start {
		data.startFrequency++
	}
	if end {
		data.endFrequency++
	}
	data.frequency++

	query := "Update Words Set Frequency = ?, EndFrequency = ?, StartFrequency = ? Where ID = ?"
	ai.BotDB.Update(query, data.frequency, data.endFrequency, data.startFrequency, id)
	return id
}

func (ai *FunAI) AddMessageToDB(message string) {
	messageArray := getWordArrayFromMessage(message)

	idArray := make([]int, len(messageArray))
	for index, word := range messageArray { //for each word in message
		data := ai.getWordInfo(word) //check if word exists in table

		id := ai.updateWord(word, index == 0, index == len(messageArray)-1, data)
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

		ai.insertIDs(insertMap) //insert values into IDs table
	}
}

//inserts word into Word table, returns ID
func (ai *FunAI) insertWord(word string) int {
	query := "insert into Words (Word, Frequency, EndFrequency, StartFrequency) values(?, ?, ?, ?)"
	ai.BotDB.Update(query, word, 0, 0, 0)
	return ai.getWordInfo(word).id
}

//inserts ids into IDs table
func (ai *FunAI) insertIDs(insertMap map[string]int) {
	query := "insert into IDs (WordID, FollowingWordID1, FollowingWordID2, FollowingWordID3, TrailingWordID1, TrailingWordID2, TrailingWordID3) values(?, ?, ?, ?, ?, ?, ?)"
	ai.BotDB.Update(query, insertMap["WordID"], insertMap["FollowingWordID1"], insertMap["FollowingWordID2"], insertMap["FollowingWordID3"], insertMap["TrailingWordID1"], insertMap["TrailingWordID2"], insertMap["TrailingWordID3"])
}

func (ai *FunAI) getSubjectWordContext(messageArray []string) wordContext {
	query := "select * from Words where Word In (?" + strings.Repeat(",?", len(messageArray)-1) + ") and Frequency > 0 order by Frequency asc"

	var params []interface{}
	for _, word := range messageArray {
		params = append(params, word)
	}

	row := ai.BotDB.QueryRow(query, params...)

	return getWordContextFromRow(row)
}

func getWordArrayFromMessage(message string) []string {
	//remove punctuation?
	message = strings.ToLower(message)
	messageArray := strings.Split(removeWhiteSpace(message), " ")
	messageArray = removeDuplicates(messageArray)
	return messageArray
}

func removeDuplicates(messageArray []string) []string {
	messageMap := make(map[string]int)
	for _, word := range messageArray {
		messageMap[word] = 1
	}
	newArray := make([]string, len(messageMap))
	index := 0
	for key, _ := range messageMap {
		newArray[index] = key
		index++
	}
	return newArray
}

func (ai *FunAI) getRandomWord() string {
	query := "SELECT Word FROM Words WHERE ID IN (SELECT ID FROM Words ORDER BY RANDOM() LIMIT 1)"

	var word string

	row := ai.BotDB.QueryRow(query)
	err := row.Scan(&word)
	if err != nil {
		return ""
	}

	return word
}

func (ai *FunAI) getIDWithSurroundingIDs(surroundingType string, ids ...int) int {
	var buffer bytes.Buffer
	var params []interface{}
	prefix := ""

	for index, id := range ids {
		buffer.WriteString(prefix)
		buffer.WriteString(fmt.Sprintf("%s%d=?", surroundingType, index+1))
		prefix = " And "

		params = append(params, id)
	}
	query := fmt.Sprintf("SELECT WordID FROM IDs WHERE %s ORDER BY RANDOM() LIMIT 1", buffer.String())

	row := ai.BotDB.QueryRow(query, params...)
	var id int
	err := row.Scan(&id)
	if err != nil {
		return 0
	}
	return id
}

func removeWhiteSpace(str string) string {
	return strings.Join(strings.Fields(str), " ")
}

func (ai *FunAI) GenerateMessage(inquiry string) string {
	inquiry = strings.Trim(inquiry, " ")
	if inquiry == "" {
		inquiry = ai.getRandomWord()
	}

	wordArray := getWordArrayFromMessage(inquiry)
	subjectWordContext := ai.getSubjectWordContext(wordArray)
	if subjectWordContext.id == 0 {
		return ""
	}

	associatedWordContext, comesAfter := ai.getAssociatedWordFromPhrase(subjectWordContext, inquiry)

	if comesAfter {
		return ai.generateStartPhrase(subjectWordContext, associatedWordContext) + ai.generateEndPhrase(subjectWordContext, associatedWordContext)
	} else {
		return ai.generateStartPhrase(associatedWordContext, subjectWordContext) + ai.generateEndPhrase(associatedWordContext, subjectWordContext)
	}
}

func (ai *FunAI) generatePhrase(surroundingType string, startWord []wordContext, phrase string) string {

	//stop condition:
	var stopFrequency int
	var frequency int
	var testWord wordContext

	for _, element := range startWord {
		testWord = element
		break
	}

	if surroundingType == "FollowingWordID" { //check for start frequency
		stopFrequency = testWord.startFrequency
		frequency = testWord.frequency
	} else {
		stopFrequency = testWord.endFrequency
		frequency = testWord.frequency
	}

	randVal := rand.Float64()
	if float64(stopFrequency)/float64(frequency) > randVal {
		fmt.Printf("Word: %s \n RandVal: %f \n StopFreq: %d \n Freq: %d\n", testWord.word, randVal, stopFrequency, frequency)
		//fmt.Printf("Phrase: %s\n", phrase)
		//fmt.Println(startWord)
		return phrase
	}

	var nextID int
	//get next word id
	if len(startWord) == 2 {
		nextID = ai.getIDWithSurroundingIDs(surroundingType, startWord[0].id, startWord[1].id)
	} else {
		nextID = ai.getIDWithSurroundingIDs(surroundingType, startWord[0].id)
	}

	if nextID == 0 {
		fmt.Println("Next id is 0")
		return phrase
	}

	//get word info
	nextwordContext := ai.getWordInfoFromID(nextID)
	//fmt.Println(fmt.Sprintf("WordID: %d Word: %s", nextwordContext.ID, nextwordContext.Word))

	//add word to phrase
	phrase = phrase + " " + nextwordContext.word

	//chain together 2 words
	nextWords := []wordContext{nextwordContext, startWord[0]}

	//recurse
	return ai.generatePhrase(surroundingType, nextWords, phrase)
}

//picks word from phrase that directly follows or preceeds subject word
//returns associated wordcontext and true if word comes after subject word (false otherwise)
func (ai *FunAI) getAssociatedWordFromPhrase(word wordContext, phrase string) (wordContext, bool) {
	wordArray := getWordArrayFromMessage(phrase)
	var query string
	var inWords []int

	var otherWords []interface{}
	if len(wordArray) > 1 {

		for _, element := range wordArray {
			if element == word.word {
				continue
			}
			id := ai.getWordInfo(element).id
			otherWords = append(otherWords, id)
			inWords = append(inWords, id)
		}

		inString := "(?" + strings.Repeat(",?", len(otherWords)-1) + ")"
		query = fmt.Sprintf("SELECT WordID, TrailingWordID1, FollowingWordID1 FROM IDs WHERE WordID=? AND (TrailingWordID1 IN %s Or FollowingWordID1 IN %s) ORDER BY RANDOM() LIMIT 1", inString, inString)
	} else {
		query = "SELECT WordID, TrailingWordID1, FollowingWordID1 FROM IDs WHERE WordID=? ORDER BY RANDOM() LIMIT 1"
	}

	//Creating new array containing all query values
	var params []interface{}
	params = append(params, word.id)
	params = append(params, otherWords...)
	params = append(params, otherWords...)

	var id int
	var prevID int
	var nextID int
	row := ai.BotDB.QueryRow(query, params...)
	err := row.Scan(&id, &prevID, &nextID)
	if err != nil {
		query = "SELECT WordID, TrailingWordID1, FollowingWordID1 FROM IDs WHERE WordID=? ORDER BY RANDOM() LIMIT 1"
		row := ai.BotDB.QueryRow(query, params[0])
		err := row.Scan(&id, &prevID, &nextID)
		if err != nil {
			return word, false
		}
	}

	if prevID != 0 && contains(inWords, prevID) {
		return ai.getWordInfoFromID(prevID), false
	} else {
		return ai.getWordInfoFromID(nextID), true
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

func (ai *FunAI) generateStartPhrase(startWord wordContext, associatedWordContext wordContext) string {
	wordContexts := []wordContext{startWord, associatedWordContext}
	if startWord == associatedWordContext {
		wordContexts = []wordContext{startWord}
	}

	phrase := reverseSentence(ai.generatePhrase("FollowingWordID", wordContexts, ""))
	return phrase
}

func (ai *FunAI) generateEndPhrase(startWord wordContext, associatedWordContext wordContext) string {
	wordContexts := []wordContext{associatedWordContext, startWord}
	if startWord == associatedWordContext {
		wordContexts = []wordContext{startWord}
	}

	phrase := ai.generatePhrase("TrailingWordID", wordContexts, wordContexts[1].word+" "+wordContexts[0].word)
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

type wordContext struct {
	word           string
	id             int
	frequency      int
	startFrequency int
	endFrequency   int
}
