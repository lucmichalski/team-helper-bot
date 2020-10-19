package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/slack-go/slack"
)

type envConfig struct {
	SlackAuthToken string `required:"true" split_words:"true"`
}

func main() {
	var env envConfig

	err := envconfig.Process("helperbot", &env)
	if err != nil {
		fmt.Println(err.Error())
	}

	dbClient, err := connectDB()
	if err != nil {
		fmt.Println("Can't create database connection")
	}
	db := newDB(dbClient)
	err = db.createTable()
	if err != nil {
		fmt.Println("Can't create helper table")
	}

	//
	err = db.getRow()
	if err != nil {
		fmt.Println("Can't get rows from helper table")
	}

	api := slack.New(
		env.SlackAuthToken,
		slack.OptionDebug(false),
		slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)),
	)

	rtm := api.NewRTM()
	s := newSlackClient(rtm, db)
	go s.slack.ManageConnection()

	for msg := range s.slack.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			msg := ev.Msg

			if msg.SubType != "" {
				break // We're only handling normal messages.
			}
			// only accept standard channel post
			direct := strings.HasPrefix(msg.Channel, "D")
			if direct {
				continue
			}

			//bot command with mention
			err := s.command(msg)
			if err != nil {
				fmt.Println("Can't execute bot command")
			}
			//catch-all reaction respons to greetings
			err = s.greetings(msg)
			if err != nil {
				fmt.Println("Can't add reaction")
			}

			//catch-all response to popular problems
			s.hellper(msg)

		case *slack.ConnectedEvent:
			fmt.Println("Connected to Slack")

		case *slack.InvalidAuthEvent:
			fmt.Println("Invalid token")
			return
		}
	}
}
