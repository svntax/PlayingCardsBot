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

// Constants for the games supported by the bot
const (
	NoGame int = iota
	HighOrLow
)

// GameState represents the current game running in a Discord server
type GameState struct {
	gameType      int
	channelID     string
	lastMessageID string
}

// ServerState holds data on the current state of a Discord server
type ServerState struct {
	id      string
	deck    playingcards.Deck
	game    GameState
	players map[string]bool
}

var serverStates = make(map[string]*ServerState)

var prefix string = "$pcb "

// NewServerState creates a new state struct for the given Discord server
func NewServerState(guildID string) *ServerState {
	ss := ServerState{id: guildID, players: make(map[string]bool)}
	return &ss
}

// GameType returns the type of game currently running in the given Discord server
func (s ServerState) GameType() int {
	return s.game.gameType
}

// Players returns a list of active (alive) and inactive(dead) players for the current game session in a Discord server
func (s *ServerState) Players() map[string]bool {
	return s.players
}

func startServer(server *http.ServeMux) {
	log.Println("Server started on port 8080")
	http.ListenAndServe(":8080", server)
}

func main() {
	rand.Seed(time.Now().Unix())

	mainServer := http.NewServeMux()
	mainServer.Handle("/", http.FileServer(http.Dir("./public")))
	mainServer.Handle("/card_images/", http.StripPrefix("/card_images/", http.FileServer(http.Dir("./card_images"))))

	go startServer(mainServer)

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("Error creating Discord session,", err)
		return
	}

	// Listen for MessageCreate events.
	dg.AddHandler(messageCreate)
	// Listen for MessageReactionAdd events
	dg.AddHandler(messageReactionAdd)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions)

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

// GetCardURL returns the full url to the image for the given card
func GetCardURL(card playingcards.Card) string {
	cardPath := GetCardPath(card)
	// URL of the server hosting the images
	hostURL := os.Getenv("HOST_URL")
	if len(hostURL) == 0 {
		hostURL = "http://localhost:8080"
	}
	cardURL := fmt.Sprintf("%s/%s", hostURL, cardPath)
	return cardURL
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

	if command == "high_or_low" {
		state := GetServerState(m.GuildID)
		if state.GameType() != NoGame {
			s.ChannelMessageSend(m.ChannelID, gameInProgressWarning())
			return
		}
		message := &discordgo.MessageEmbed{
			Color:       0x3dbb6b,
			Title:       "High or Low",
			Description: "Guess whether the next card will be higher or lower.\nReact with your choice.",
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Game starting in 5 seconds...",
			},
		}
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, message)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error when trying to start the game.")
			return
		}
		go HighOrLowGame(state, s, m.ChannelID)
	}

	if command == "draw" {
		state := GetServerState(m.GuildID)
		if state.GameType() != NoGame {
			s.ChannelMessageSend(m.ChannelID, gameInProgressWarning())
			return
		}

		cardDrawn := state.deck.DrawCard()
		if strings.Contains(cardDrawn.String(), "Invalid") {
			s.ChannelMessageSend(m.ChannelID, "No more cards left!")
		} else {
			cardURL := GetCardURL(cardDrawn)
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
		if state.GameType() != NoGame {
			s.ChannelMessageSend(m.ChannelID, gameInProgressWarning())
			return
		}

		state.deck.Shuffle()
		s.ChannelMessageSend(m.ChannelID, "Cards shuffled!")
	}
	if command == "reset_cards" {
		state := GetServerState(m.GuildID)
		if state.GameType() != NoGame {
			s.ChannelMessageSend(m.ChannelID, gameInProgressWarning())
			return
		}

		state.deck = playingcards.NewDeck()
		s.ChannelMessageSend(m.ChannelID, "Cards have been reset.")
	}
	if command == "quitgame" {
		state := GetServerState(m.GuildID)
		if state.GameType() == NoGame {
			s.ChannelMessageSend(m.ChannelID, "There is no game in progress.")
			return
		}

		state.game.gameType = NoGame
		state.deck = playingcards.NewDeck()
		s.ChannelMessageSend(m.ChannelID, "Stopped the game.")
	}
}

func gameInProgressWarning() string {
	return fmt.Sprintf("A game is currently in progress! Enter %squitgame to stop the game.", prefix)
}

func messageReactionAdd(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	state := GetServerState(m.GuildID)
	if state.GameType() == NoGame {
		// Reactions are used in games only
		return
	}
	if m.MessageReaction.UserID == s.State.User.ID || state.game.channelID != m.ChannelID {
		return
	}

	if state.game.lastMessageID == m.MessageID {
		reactionName := m.MessageReaction.Emoji.APIName()
		if reactionName == "⬆️" {
			fmt.Println("High")
		} else if reactionName == "⬇️" {
			fmt.Println("Low")
		}
		// TODO: put players into 2 groups for high-low guess
		state.Players()[m.MessageReaction.UserID] = true
	}
}

// HighOrLowGame starts a new game of High or Low for the given Discord server in the channel the bot responded to.
func HighOrLowGame(state *ServerState, s *discordgo.Session, channelID string) {
	state.game.gameType = HighOrLow
	state.deck = playingcards.NewDeck()
	state.deck.Shuffle()
	time.Sleep(5 * time.Second)

	// Set up the game state
	state.game.channelID = channelID

	// Game loop
	for {
		if state.game.gameType != HighOrLow {
			// Reset game state and leave
			state.game.gameType = NoGame
			state.game.channelID = ""
			state.players = make(map[string]bool)
			state.deck = playingcards.NewDeck()
			return
		}
		cardDrawn := state.deck.DrawCard()
		if strings.Contains(cardDrawn.String(), "Invalid") {
			s.ChannelMessageSend(channelID, "No more cards left!")
			break
		} else {
			// First draw a card, add up/down reactions, then wait 5 seconds
			cardURL := GetCardURL(cardDrawn)
			message := &discordgo.MessageEmbed{
				Color: 0x3dbb6b,
				Title: cardDrawn.String(),
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("%d cards remaining.", state.deck.Size()),
				},
				Image: &discordgo.MessageEmbedImage{
					URL: cardURL,
				},
			}
			messageObj, err := s.ChannelMessageSendEmbed(channelID, message)
			if err != nil {
				// Something went wrong, stop the game
				state.game.gameType = NoGame
				state.deck = playingcards.NewDeck()
				s.ChannelMessageSend(channelID, "Error found while running the game. Exiting...")
				return
			}
			s.MessageReactionAdd(channelID, messageObj.ID, "\xe2\xac\x86\xef\xb8\x8f")
			s.MessageReactionAdd(channelID, messageObj.ID, "\xe2\xac\x87\xef\xb8\x8f")
			state.game.lastMessageID = messageObj.ID

			time.Sleep(5 * time.Second)

			// TODO Check all players who have reacted, remove wrong responses
			break
		}
	}

	// List the winners
	winners := "Winners: "
	for player, won := range state.Players() {
		if won {
			member, err := s.GuildMember(state.id, player)
			if err != nil {
				continue
			}
			playerName := member.Nick
			if len(playerName) == 0 {
				playerName = member.User.Username
			}
			winners += fmt.Sprintf("%s ", playerName)
		}
	}

	s.ChannelMessageSend(channelID, winners)

	// Reset game state
	state.game.gameType = NoGame
	state.game.channelID = ""
	state.players = make(map[string]bool)
	state.deck = playingcards.NewDeck()
}
