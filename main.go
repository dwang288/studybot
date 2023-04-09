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

func main() {
	tokenID := flag.String("bot-token", "", "Discord bot token")
	userID := flag.String("user-id", "", "User ID")

	flag.Parse()

	err := godotenv.Load(getAbsolutePath("env/secrets.env"))
	checkErr(err)

	// Create a new Discord session using the bot token
	dg, err := discordgo.New("Bot " + *tokenID)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	arr := []string{
		"is this a game to you?",
		"you're not somebody.",
		"don't you have work to do?",
		"i didn't know this was more important than patient care?",
		"why are you still here? i will CHEW your MEAT",
		"sasuga exam failer.",
		"tell me what you've learned in the past three hours",
		"damn so if i give you the test right now it's just gonna be flying colors right?",
	}

	// Every 30 min add token, initially allows a burst of 5
	limiter := rate.NewLimiter(0.0005, 5)
	dg.AddHandler(messageCreate(dg, arr, *userID, limiter))

	// Open a connection to Discord
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening Discord connection:", err)
		return
	}

	// Wait for a CTRL-C signal to close the bot
	fmt.Println("Bot is running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Close the Discord session
	dg.Close()
}

// This function is called whenever a new message is created
func messageCreate(s *discordgo.Session, arr []string, userID string, limiter *rate.Limiter) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Check if the message was sent by the user we want to monitor

	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Check if the message is not from a bot (to avoid sending messages from the bot to the channel)
		if m.Author.ID == userID && !m.Author.Bot && limiter.Allow() {
			// Send a message to the channel
			phrase := randomPhrase(arr)
			_, err := s.ChannelMessageSend(m.ChannelID, phrase)
			log.Printf("m.Author.ID: %v, m.ChannelID: %v, phrase: %v", m.Author.ID, m.ChannelID, phrase)
			if err != nil {
				fmt.Println("Error sending message to channel:", err)
			}
		}
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

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
