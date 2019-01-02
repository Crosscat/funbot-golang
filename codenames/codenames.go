package codenames

import (
	"fmt"
	"funbot/utils"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	wordFile string
	adminID  string

	redTeam            map[string]int
	blueTeam           map[string]int
	userIDMap          map[string]string
	redTeamCodeMaster  string
	blueTeamCodeMaster string
	startingTeam       int
	currentTurn        int
	guessesRemaining   int
	hintGiven          bool

	boardSize  int
	words      []string
	goal       []int
	goalMap    map[string]int
	wordBank   []string
	goalRed    map[string]int
	goalBlue   map[string]int
	neutralMap map[string]int
	assassin   string
)

func Setup(wordBankFileName string, adminId string) {
	wordFile = wordBankFileName
	adminID = adminId
}

func ProcessCommand(m *discordgo.MessageCreate, s *discordgo.Session) string {
	message := m.Content
	userName := m.Author.Username
	//userID := m.Author.ID what was this for???

	message = strings.ToLower(strings.TrimSpace(message[3:len(message)]))
	splitMessage := strings.Split(message, " ")

	switch splitMessage[0] {

	case "init", "reset":
		//if !isAdmin(userID) {
		//	return "Only admin can perform that command right now."
		//}
		initialize(wordFile)
		return "Let's play codenames!"

	case "start":
		//if !isAdmin(userID) {
		//	return "Only admin can perform that command right now."
		//}
		if len(redTeam) < 2 || len(blueTeam) < 2 {
			return "Not enough players to start!"
		}
		initializeGame(boardSize)
		sendGoalToPlayer(userIDMap[redTeamCodeMaster], s)
		sendGoalToPlayer(userIDMap[blueTeamCodeMaster], s)
		return fmt.Sprintf("Starting new game!\n\n%s\n%s is Red team code master!\n\n%s is Blue team code master!\n\n%s team goes first!\n\n%s", getTeams(), redTeamCodeMaster, blueTeamCodeMaster, getTeamName(currentTurn), getDisplayableBoard(boardSize))

	case "join":
		if currentTurn > -1 {
			return fmt.Sprintf("%s: Can't change teams while game is in progress!", userName)
		}
		var team int
		if len(splitMessage) > 1 {
			team = addPlayerS(userName, splitMessage[1])
		} else {
			team = addPlayer(userName, 2)
		}
		userIDMap[userName] = m.Author.ID
		return fmt.Sprintf("%s has joined team %s!", userName, getTeamName(team))

	case "show", "list":
		return getDisplayableBoard(boardSize)

	case "hint":
		var currentCodeMaster string
		switch currentTurn {
		case 0:
			currentCodeMaster = redTeamCodeMaster
		case 1:
			currentCodeMaster = blueTeamCodeMaster
		}
		if currentCodeMaster != userName {
			return fmt.Sprintf("Only %s can give hints right now!", currentCodeMaster)
		}
		if len(splitMessage) > 2 {
			guessesRemaining, _ = strconv.Atoi(splitMessage[2])
			guessesRemaining++
		} else {
			guessesRemaining = 99999
		}
		hintGiven = true
		return fmt.Sprintf("Hint has been given! Guesses left: %d", guessesRemaining)

	case "guess":
		if len(splitMessage) != 2 {
			return fmt.Sprintf("%s: Invalid guess.", userName)
		}
		return guess(splitMessage[1], userName)

	case "rules":
		return getRules()

	case "cm":
		team := getTeam(userName)
		if team == -1 {
			return fmt.Sprintf("%s: Join a team first!")
		} else if team == 0 {
			redTeamCodeMaster = userName
		} else if team == 1 {
			blueTeamCodeMaster = userName
		}
		return fmt.Sprintf("%s is now code master for %s team!", userName, getTeamName(team))
	}

	return fmt.Sprintf("%s: Not really sure what you mean.", userName)
}

func getTeam(player string) int {
	if _, contains := redTeam[player]; contains {
		return 0
	}
	if _, contains := blueTeam[player]; contains {
		return 1
	}
	return -1
}

func initialize(wordListFile string) {
	rand.Seed(time.Now().UTC().UnixNano())
	currentTurn = -1
	boardSize = 5
	if wordListFile != "" {
		wordBank, _ = utils.ReadLines(wordListFile)
	}
	redTeam = make(map[string]int)
	blueTeam = make(map[string]int)
	userIDMap = make(map[string]string)
	hintGiven = false
}

func initializeGame(size int) {
	setCodeMasters()
	currentTurn = rand.Intn(2)
	startingTeam = currentTurn
	words = getWordBoard(wordBank, size)
	initializeGoalBoard(size)
	shuffleGoalBoard()
}

func setCodeMasters() {
	if redTeamCodeMaster == "" {
		redTeamCodeMaster = getRandomKeyFromMap(redTeam)
	}
	if blueTeamCodeMaster == "" {
		blueTeamCodeMaster = getRandomKeyFromMap(blueTeam)
	}
}

func getRandomKeyFromMap(m map[string]int) string {
	i := rand.Intn(len(m))
	for k := range m {
		if i == 0 {
			return k
		}
		i--
	}
	return ""
}

func getGoal() string {
	message := "Red:\n"
	for word, _ := range goalRed {
		message += word + "\n"
	}
	message += "\nBlue:\n"
	for word, _ := range goalBlue {
		message += word + "\n"
	}
	message += "\nNeutral:\n"
	for word, _ := range neutralMap {
		message += word + "\n"
	}
	message += "\nAssassin:\n" + assassin

	return message
}

func initializeGoalBoard(size int) {
	goal = []int{-1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 3}
}

func getTeamName(num int) string {
	switch num {
	case 0:
		return "Red"
	case 1:
		return "Blue"
	}
	return "Red"
}

func addPlayerS(playerID string, color string) int {
	switch strings.ToLower(color) {
	case "red":
		return addPlayer(playerID, 0)
	case "blue":
		return addPlayer(playerID, 1)
	default:
		return addPlayer(playerID, 2)
	}
}

func addPlayer(playerID string, team int) int {
	switch team {
	case 0: // red team
		if _, contains := blueTeam[playerID]; contains {
			delete(blueTeam, playerID)
		}
		redTeam[playerID] = team
	case 1: // blue team
		if _, contains := redTeam[playerID]; contains {
			delete(redTeam, playerID)
		}
		blueTeam[playerID] = team
	case 2: // whichever team has fewer members
		if len(redTeam) > len(blueTeam) {
			return addPlayer(playerID, 1)
		} else {
			return addPlayer(playerID, 0)
		}
	}
	return team
}

func guess(guess string, guesser string) string {
	if !hintGiven {
		return "Hint has not been given yet!"
	}

	var currentTeam *map[string]int
	var teamGoal *map[string]int

	if currentTurn == 0 {
		currentTeam = &redTeam
		teamGoal = &goalRed
	} else {
		currentTeam = &blueTeam
		teamGoal = &goalBlue
	}
	if _, exists := (*currentTeam)[guesser]; !exists {
		return fmt.Sprintf("%s is not allowed to guess right now!", guesser)
	}

	if guess == "skip" {
		return endTurn()
	}

	if !utils.ArrayContainsNoCase(words, guess) {
		return fmt.Sprintf("%s is not available for guessing!", guess)
	}

	goalValue := showWord(guess)
	returnMessage := ""

	switch goalValue {
	case 0:
		delete(goalRed, guess)
		returnMessage += getDisplayableBoard(boardSize) + "\nRed team gets the point!"
	case 1:
		delete(goalBlue, guess)
		returnMessage += getDisplayableBoard(boardSize) + "\nBlue team gets the point!"
	case 2:
		returnMessage += "Neither team gets the point!"
	case 3:
		returnMessage += fmt.Sprintf("Assassinated! %s team is the winner!", getTeamName((currentTurn+1)%2))
		endGame()
		return returnMessage
	}

	if len(*teamGoal) == 0 {
		return fmt.Sprintf("%s\nCongratulations! %s team is the winner!", returnMessage, getTeamName(currentTurn))
	}

	if currentTurn == goalValue {
		guessesRemaining--
		if guessesRemaining > 0 {
			return fmt.Sprintf("%s Guesses left: %d", returnMessage, guessesRemaining)
		} else {
			return fmt.Sprintf("%s\n%s", returnMessage, endTurn())
		}
	} else {
		return fmt.Sprintf("%s\n%s", returnMessage, endTurn())
	}
}

func showWord(guess string) int {
	goalValue := goalMap[guess]
	//newWord := goalValueToString(goalValue)
	replaceWord(guess, "")
	return goalValue
}

func goalValueToString(goalValue int) string {
	switch goalValue {
	case 0:
		return "RED"
	case 1:
		return "BLUE"
	case 2:
		return "EMPTY"
	case 3:
		return "ASSASSIN"
	}
	return ""
}

func replaceWord(word string, newWord string) {
	for i := 0; i < len(words); i++ {
		if words[i] == word {
			words[i] = newWord
			return
		}
	}
}

func endTurn() string {
	hintGiven = false
	currentTurn = (currentTurn + 1) % 2
	return fmt.Sprintf("Ending turn, it is now %s team's turn!", getTeamName(currentTurn))
}

func endGame() {
	redTeamCodeMaster = ""
	blueTeamCodeMaster = ""
	initialize("")
}

func getWordBoard(wordBank []string, size int) []string {
	wordBoard := make([]string, size*size)
	usedIndices := make(map[int]int)
	index := 0
	for index < size*size {
		randNum := rand.Intn(len(wordBank))
		if _, exists := usedIndices[randNum]; !exists {
			wordBoard[index] = wordBank[randNum]
			usedIndices[randNum] = randNum
			index++
		}
	}

	return wordBoard
}

func shuffleGoalBoard() {
	goal[0] = currentTurn
	for i := range goal {
		j := rand.Intn(i + 1)
		goal[i], goal[j] = goal[j], goal[i]
	}

	goalMap = make(map[string]int)
	goalRed = make(map[string]int)
	goalBlue = make(map[string]int)
	neutralMap = make(map[string]int)

	for index, value := range goal {
		goalMap[words[index]] = value
		switch value {
		case 0:
			goalRed[words[index]] = 1
		case 1:
			goalBlue[words[index]] = 1
		case 2:
			neutralMap[words[index]] = 1
		case 3:
			assassin = words[index]
		}
	}
}

func getScore() string {
	totalRed := 8
	totalBlue := 8
	if startingTeam == 0 {
		totalRed++
	} else {
		totalBlue++
	}
	return fmt.Sprintf("Red team: %d/%d\nBlue team: %d/%d", totalRed-len(goalRed), totalRed, totalBlue-len(goalBlue), totalBlue)
}

func getTeams() string {
	redTeamString := "RED TEAM:\n"
	for key, _ := range redTeam {
		redTeamString += key + "\n"
	}
	blueTeamString := "BLUE TEAM:\n"
	for key, _ := range blueTeam {
		blueTeamString += key + "\n"
	}
	return fmt.Sprintf("%s\n%s", redTeamString, blueTeamString)
}

func getDisplayableBoard(size int) string {
	longestLength := 20
	formattedBoard := "```"
	for index, word := range words {
		formattedBoard += word + strings.Repeat(" ", longestLength-len(word))
		if index%size == size-1 {
			formattedBoard += "\n"
		}
	}
	return getScore() + "\n" + formattedBoard + "```"
}

func isAdmin(userID string) bool {
	return userID == adminID
}

func sendGoalToPlayer(userID string, s *discordgo.Session) {
	privateChannel, err := s.UserChannelCreate(userID)
	if err != nil {
		fmt.Println("Failed to send message to user " + userID)
	}
	s.ChannelMessageSend(privateChannel.ID, "Congratulations, you are a code master!  Here is what you need to know:\n\n"+getGoal())
}

func getRules() string {
	return `In this game there are two teams whose goal is to guess the correct words based on limited information.
One player from each team will be a "codemaster" who will know which word belongs to which team.
The code masters will take turns giving a single-word hint (along with the number of words matching the hint) to point to their team's words while avoiding having their team guess the other team's words.
Additionally, any team that guesses the assassin word will immediately lose.  First team to guess all of their words wins!
Teams can guess up to n+1 times, where n is the number of words given by the code master.
	
Commands:
!cn join [red/blue] to join a team
!cn hint [word] [number of guesses] for the codemaster to give a hint
!cn guess [word] for the current team to guess a word`
}
