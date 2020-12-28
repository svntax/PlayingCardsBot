package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/svntax/PlayingCardsBot/playingcards"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Bot token is passed as a command line argument
var (
	Token string
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
}

// ServerState holds data on the current state of a Discord server
type ServerState struct {
	id             string
	deck           playingcards.Deck
	gameInProgress bool
}

var serverStates = make(map[string]*ServerState)

// NewServerState creates a new state struct for the given Discord server
func NewServerState(guildID string) *ServerState {
	ss := ServerState{id: guildID}
	return &ss
}

func main() {
	rand.Seed(time.Now().Unix())

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("Error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()

	/*
		deck := playingcards.NewDeck()
		deck.Shuffle()
		for i := 1; i <= 53; i++ {
			card := deck.DrawCard()
			fmt.Println(card)
		}
		deck.Shuffle()
		fmt.Println(deck.DrawCard())

		deck = playingcards.NewDeck()
		for i := 1; i <= 52; i++ {
			fmt.Println(deck.DrawCard())
		}
	*/
}

// GetServerState looks for the given server and returns it if it exists, or creates a new entry first
func GetServerState(guildID string) *ServerState {
	state, exists := serverStates[guildID]
	if !exists {
		fmt.Println("New guild:", guildID)
		// Add the server to the list of servers
		state = NewServerState(guildID)
		serverStates[guildID] = state
		// Initialize the server's deck of cards
		state.deck = playingcards.NewDeck()
	}
	return state
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.Content == "info" {
		s.ChannelMessageSend(m.ChannelID, "This bot allows users to play with a standard 52-card deck of playing cards.")
	}

	if m.Content == "draw" {
		state := GetServerState(m.GuildID)
		cardDrawn := state.deck.DrawCard()
		s.ChannelMessageSend(m.ChannelID, cardDrawn.String())
	}
	if m.Content == "shuffle" {
		state := GetServerState(m.GuildID)
		state.deck.Shuffle()
		s.ChannelMessageSend(m.ChannelID, "Cards shuffled!")
	}
	if m.Content == "reset_cards" {
		state := GetServerState(m.GuildID)
		state.deck = playingcards.NewDeck()
		s.ChannelMessageSend(m.ChannelID, "Cards have been reset.")
	}
}
