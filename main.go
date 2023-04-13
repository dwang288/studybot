package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/time/rate"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type Config struct {
	UserID                 string
	InactiveTimerangeStart time.Time
	InactiveTimerangeEnd   time.Time
	Phrases                []string
}

func (config *Config) isWithinTheTimePeriod(t time.Time) bool {
	return config.InactiveTimerangeStart.Before(t) && config.InactiveTimerangeEnd.After(t)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {

	config := Config{
		UserID:                 "",
		InactiveTimerangeStart: time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC), // default value is off
		InactiveTimerangeEnd:   time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC),
		Phrases: []string{
			"is this a game to you?",
			"you're not somebody.",
			"don't you have work to do?",
			"i didn't know this was more important than patient care?",
			"why are you still here? i will CHEW your MEAT",
			"sasuga exam failer.",
			"tell me what you've learned in the past three hours",
			"damn so if i give you the test right now it's just gonna be flying colors right?",
		},
	}

	err := godotenv.Load(getAbsolutePath("env/secrets.env"))
	if err != nil {
		log.Fatalln("Error opening secrets file:", err)
	}

	uid := flag.String("user-id", "", "User ID")
	flag.Parse()

	config.UserID = *uid
	if config.UserID == "" {
		config.UserID = os.Getenv("USER_ID")
	}

	dg, err := discordgo.New("Bot " + os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatalln("Error creating Discord session:", err)
	}

	// Open a connection to Discord
	err = dg.Open()
	if err != nil {
		log.Fatalln("Error opening Discord connection:", err)
	}

	addCommands(dg)

	// Every 30 min add token, initially allows a burst of 5
	limiter := rate.NewLimiter(0.0005, 5)
	dg.AddHandler(messageCreate(dg, &config, limiter))
	dg.AddHandler(setTime(dg, &config))

	// Wait for a CTRL-C signal to close the bot
	fmt.Println("Bot is running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Close the Discord session
	dg.Close()
}

func addCommands(dg *discordgo.Session) {
	command := &discordgo.ApplicationCommand{
		Name:        "set_duration",
		Description: "syntax: set_duration minutes:<minutes>",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "minutes",
				Description: "the duration in minutes the bot should be disabled for",
				Required:    true,
			},
		},
	}
	log.Printf("command: %v, dg.State.User.ID: %v", command, dg.State.User.ID)
	_, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", command)

	if err != nil {
		log.Println("Error creating application command:", err)
	}
}

// This function is called whenever a new message is created
func messageCreate(s *discordgo.Session, config *Config, limiter *rate.Limiter) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Check if the message was sent by the user we want to monitor

	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Check if the message is not from a bot (to avoid sending messages from the bot to the channel)
		if m.Author.ID == config.UserID && !m.Author.Bot && !config.isWithinTheTimePeriod(time.Now()) && limiter.Allow() {
			// Send a message to the channel
			phrase := randomPhrase(config.Phrases)
			_, err := s.ChannelMessageSend(m.ChannelID, phrase)
			log.Printf("m.Author.ID: %v, m.ChannelID: %v, phrase: %v", m.Author.ID, m.ChannelID, phrase)
			if err != nil {
				fmt.Println("Error sending message to channel:", err)
			}
		}
	}
}

func setTime(s *discordgo.Session, config *Config) func(s *discordgo.Session, i *discordgo.InteractionCreate) {

	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {

		var responseString string

		if i.ApplicationCommandData().Name != "set_duration" && i.Member != nil && i.Member.User.ID != config.UserID {
			return
		}

		for _, opt := range i.ApplicationCommandData().Options {
			switch opt.Name {
			case "minutes":
				intervalMinutes := time.Duration(i.ApplicationCommandData().Options[0].IntValue()) * time.Minute
				config.InactiveTimerangeStart = time.Now()
				config.InactiveTimerangeEnd = config.InactiveTimerangeStart.Add(intervalMinutes)

				responseString = fmt.Sprintf("free period set from %s to %s",
					config.InactiveTimerangeStart.Format("15:04PM"),
					config.InactiveTimerangeEnd.Format("15:04PM"),
				)
			default:
				responseString = "wrong format"
			}

		}

		log.Printf("start: %s\nend: %s", config.InactiveTimerangeStart, config.InactiveTimerangeEnd)

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: responseString,
			},
		})
	}

}

func randomPhrase(arr []string) string {
	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(len(arr))
	return arr[i]
}

func getAbsolutePath(path string) string {
	execPath, err := os.Executable()
	checkErr(err)
	return filepath.Join(filepath.Dir(execPath), path)
}
