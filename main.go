package main

import (
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/svntax/PlayingCardsBot/playingcards"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Bot token can be passed as a command line argument
var (
	Token string
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.Parse()
	flagFound := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "t" {
			if len(f.Value.String()) > 0 {
				flagFound = true
			}
		}
	})
	if !flagFound {
		// Bot token is read as an environment variable if no command line argument was found
		Token = os.Getenv("BOT_TOKEN")
	}
}

// ServerState holds data on the current state of a Discord server
type ServerState struct {
	id             string
	deck           playingcards.Deck
	gameInProgress bool
}

var serverStates = make(map[string]*ServerState)

var prefix string = "$pcb "

// NewServerState creates a new state struct for the given Discord server
func NewServerState(guildID string) *ServerState {
	ss := ServerState{id: guildID}
	return &ss
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Playing cards bot server.")
}

func startServer(server *http.ServeMux) {
	log.Println("Server started on port 8080")
	http.ListenAndServe(":8080", server)
}

func main() {
	rand.Seed(time.Now().Unix())

	mainServer := http.NewServeMux()
	mainServer.HandleFunc("/", mainHandler)
	mainServer.Handle("/card_images/", http.StripPrefix("/card_images/", http.FileServer(http.Dir("./card_images"))))

	go startServer(mainServer)

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
}

// GetServerState looks for the given server and returns it if it exists, or creates a new entry first
func GetServerState(guildID string) *ServerState {
	state, exists := serverStates[guildID]
	if !exists {
		// Add the server to the list of servers
		state = NewServerState(guildID)
		serverStates[guildID] = state
		// Initialize the server's deck of cards
		state.deck = playingcards.NewDeck()
	}
	return state
}

// GetCardPath is specific to how the card images are named
func GetCardPath(card playingcards.Card) string {
	// Kenney's cards are of the format "card_<suit>_XX"
	suit := strings.ToLower(card.Suit().String())
	valueString := ""
	switch card.Value() {
	case 1:
		valueString = "A"
	case 2, 3, 4, 5, 6, 7, 8, 9, 10:
		valueString = fmt.Sprintf("%02d", card.Value())
	case 11:
		valueString = "J"
	case 12:
		valueString = "Q"
	case 13:
		valueString = "K"
	default:
		return ""
	}
	path := fmt.Sprintf("card_images/kenney_cards_large/card_%s_%s.png", suit, valueString)
	return path
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}
	// Check for the prefix string
	if !strings.HasPrefix(m.Content, prefix) {
		return
	}
	command := strings.TrimPrefix(m.Content, prefix)

	if command == "info" {
		s.ChannelMessageSend(m.ChannelID, "This bot allows users to play with a standard 52-card deck of playing cards.")
	}

	if command == "draw" {
		state := GetServerState(m.GuildID)
		cardDrawn := state.deck.DrawCard()
		if strings.Contains(cardDrawn.String(), "Invalid") {
			s.ChannelMessageSend(m.ChannelID, "No more cards left!")
		} else {
			cardPath := GetCardPath(cardDrawn)
			// URL of the server hosting the images
			hostURL := os.Getenv("HOST_URL")
			if len(hostURL) == 0 {
				hostURL = "http://localhost"
			}
			cardURL := fmt.Sprintf("%s:8080/%s", hostURL, cardPath)
			message := &discordgo.MessageEmbed{
				Color: 0x7fb2f0,
				Title: cardDrawn.String(),
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("%d cards remaining.", state.deck.Size()),
				},
				Image: &discordgo.MessageEmbedImage{
					URL: cardURL,
				},
			}
			s.ChannelMessageSendEmbed(m.ChannelID, message)
		}
	}
	if command == "shuffle" {
		state := GetServerState(m.GuildID)
		state.deck.Shuffle()
		s.ChannelMessageSend(m.ChannelID, "Cards shuffled!")
	}
	if command == "reset_cards" {
		state := GetServerState(m.GuildID)
		state.deck = playingcards.NewDeck()
		s.ChannelMessageSend(m.ChannelID, "Cards have been reset.")
	}
}
