package reddit

import (
	"fmt"
	"funbot/db"
	"strings"

	"github.com/turnage/graw/reddit"
)

var (
	botDB     *db.BotDB
	redditBot reddit.Bot
)

func Initialize(db *db.BotDB, agentFileName string) {
	botDB = db
	redditBot = SetupBot(agentFileName)
}

func SetupBot(agentFileName string) reddit.Bot {
	bot, _ := reddit.NewBotFromAgentFile(agentFileName, 0)
	return bot
}

func GetRandomTopSubPost(sub string, includeNSFW bool) string {

	if !strings.HasPrefix(sub, "/r/") {
		sub = "/r/" + sub
	}

	harvest, err := redditBot.Listing(sub, "")
	if err != nil || len(harvest.Posts) == 0 {
		fmt.Printf("Failed to fetch from sub %s\n", sub)
		return "I can't find anything on that sub"
	}

	for _, post := range harvest.Posts[:len(harvest.Posts)] {
		if post.IsSelf {
			continue
		}

		if !includeNSFW && post.NSFW {
			continue
		}

		if post.Stickied {
			continue
		}

		url := post.URL
		alreadyLinked := urlAlreadyLinked(url)
		if alreadyLinked {
			continue
		}

		AddURLToDB(url)
		return url
	}

	return "I'm all out of fresh content"
}

func urlAlreadyLinked(url string) bool {
	row := botDB.QueryRow("select * from Urls where URL = ?", url)
	var val string
	err := row.Scan(&val)
	if err != nil {
		return false
	}
	return true
}

func AddURLToDB(url string) {
	botDB.Update("Insert into Urls (URL) values (?)", url)
}
