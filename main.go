package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/svntax/PlayingCardsBot/playingcards"
)

// Bot token can be passed as a command line argument
var (
	Token          string
	GuildID        = flag.String("guild", "", "Test guild ID. If not passed - bot registers commands globally")
	AppID          string
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&AppID, "app", "", "Application ID")
	flag.Parse()
	tokenFound := false
	appIDFound := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "t" {
			if len(f.Value.String()) > 0 {
				tokenFound = true
			}
		}
		if f.Name == "app" {
			if len(f.Value.String()) > 0 {
				appIDFound = true
			}
		}
	})
	if !tokenFound {
		// Bot token is read as an environment variable if no command line argument was found
		Token = os.Getenv("BOT_TOKEN")
	}
	if !appIDFound {
		AppID = os.Getenv("APPLICATION_ID")
	}
}

// Constants for the games supported by the bot
const (
	NoGame int = iota
	HighOrLow
)

// Constants that represent a player's decision in a High or Low game
const (
	NoGuess int = iota
	High
	Low
)

// PlayerState represents a player's state during a game of High or Low
type PlayerState struct {
	choice int
	active bool
}

// Active returns whether a player is still in the currently running game or not
func (p PlayerState) Active() bool {
	return p.active
}

// GameState represents the current game running in a Discord server
type GameState struct {
	gameType      int
	channelID     string
	lastMessageID string
	preStartPhase bool
}

// ServerState holds data on the current state of a Discord server
type ServerState struct {
	id            string
	deck          playingcards.Deck
	game          GameState
	players       map[string]*PlayerState
	cardsStyle    int
	includeJokers bool
}

// Constants that represent what card images to use
const (
	KenneyLarge = iota
	KenneyPixel
)

var serverStates = make(map[string]*ServerState)

var prefix string = "$pcb "

// NewServerState creates a new state struct for the given Discord server
func NewServerState(guildID string) *ServerState {
	ss := ServerState{id: guildID, players: make(map[string]*PlayerState), cardsStyle: KenneyLarge, includeJokers: false}
	return &ss
}

// GameType returns the type of game currently running in the given Discord server
func (s ServerState) GameType() int {
	return s.game.gameType
}

// Players returns a list of active (alive) and inactive(dead) players for the current game session in a Discord server
func (s *ServerState) Players() map[string]*PlayerState {
	return s.players
}

func startServer(server *http.ServeMux) {
	log.Println("Server started on port 8080")
	http.ListenAndServe(":8080", server)
}

// Slash commands setup
var (
	integerOptionMinValue          = 1.0
	dmPermission                   = false
	defaultMemberPermissions int64 = discordgo.PermissionManageServer

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "info",
			Description: "List all commands available.",
		},
		{
			Name:        "shuffle",
			Description: "Shuffle the cards in the deck.",
		},
		{
			Name:        "reset-cards",
			Description: "Reset the deck of cards as if it were brand new.",
		},
		{
			Name:        "quit-game",
			Description: "Stop any currently running game.",
		},
		{
			Name:        "draw",
			Description: "Draw a card from the deck.", // TODO: integer option to draw multiple cards
		},
		{
			Name:        "set-style",
			Description: "Change the art style of the cards.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "style",
					Description: "The name of the art style (\"normal\", \"pixel\")",
					Required:    true,
				},
			},
		},
		{
			Name:        "include-jokers",
			Description: "Set whether the Joker cards should be in the deck or not.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "include-jokers",
					Description: "Include the Joker cards?",
					Required:    true,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"info": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			message := &discordgo.MessageEmbed{
				Color:       0x607d8b,
				Title:       "Playing Cards Bot Info",
				Description: getInfoText(),
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{message},
				},
			})
		},
		"shuffle": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			msg := ""
			state := GetServerState(i.GuildID)
			if state.GameType() != NoGame {
				msg = gameInProgressWarning()
			} else {
				state.deck.Shuffle()
				msg = "Cards shuffled!"
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msg,
				},
			})
		},
		"reset-cards": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			msg := ""
			state := GetServerState(i.GuildID)
			if state.GameType() != NoGame {
				msg = gameInProgressWarning()
			} else {
				state.deck = playingcards.NewDeck(state.includeJokers)
				msg = "Cards have been reset."
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msg,
				},
			})
		},
		"draw": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			state := GetServerState(i.GuildID)
			if state.GameType() != NoGame {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: gameInProgressWarning(),
					},
				})
			} else {
				cardDrawn := state.deck.DrawCard()
				if strings.Contains(cardDrawn.String(), "Invalid") {
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "No more cards left!",
						},
					})
				} else {
					cardURL := GetCardURL(cardDrawn, state.cardsStyle)
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
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Embeds: []*discordgo.MessageEmbed{message},
						},
					})
				}
			}
		},
		"quit-game": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			msg := ""
			state := GetServerState(i.GuildID)
			if state.GameType() == NoGame {
				msg = "There is no game in progress."
			} else {
				state.game.gameType = NoGame
				state.deck = playingcards.NewDeck(state.includeJokers)
				msg = "Stopped the game."
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msg,
				},
			})
		},
		"set-style": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			msg := "No change was made."
			if option, ok := optionMap["style"]; ok {
				style := strings.ToLower(option.StringValue())
				msg = setCardsStyle(i.GuildID, style)
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msg,
				},
			})
		},
		"include-jokers": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			msg := "No change was made."
			if option, ok := optionMap["include-jokers"]; ok {
				msg = toggleJokerCards(i.GuildID, option.BoolValue())
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msg,
				},
			})
		},
	}
)

// End of slash commands setup

func getInfoText() string {
	// Copied from message listener
	var infoString strings.Builder
	infoString.WriteString("This bot allows users to play with a standard 52-card deck of playing cards.\n\n")
	infoString.WriteString(fmt.Sprintf("**/draw**, **%sdraw**: Draw a card from the current deck.\n", prefix))
	infoString.WriteString(fmt.Sprintf("**/shuffle**, **%sshuffle**: Shuffle the current deck of cards.\n", prefix))
	infoString.WriteString(fmt.Sprintf("**/reset-cards**, **%sreset_cards**: Make a brand new, ordered deck of 52 cards.\n", prefix))
	infoString.WriteString("**/set-style**: Change the style of the cards. Options are \"normal\" and \"pixel\".\n")
	infoString.WriteString(fmt.Sprintf("**%sset_style_normal**: Change the style of the cards to normal.\n", prefix))
	infoString.WriteString(fmt.Sprintf("**%sset_style_pixel**: Change the style of the cards to pixel art.\n", prefix))
	infoString.WriteString("**/include-jokers**: Add or remove the Joker cards from the deck.\n")
	infoString.WriteString(fmt.Sprintf("**%sinclude_jokers**: Add the red and black Joker cards to the deck.\n", prefix))
	infoString.WriteString(fmt.Sprintf("**%sremove_jokers**: Remove the red and black Joker cards from the deck.\n", prefix))

	infoString.WriteString("\n__**Games**__\n")
	infoString.WriteString(fmt.Sprintf("**%shigh_or_low**: Start a game of High or Low.\n", prefix))
	infoString.WriteString(fmt.Sprintf("**/quit-game**, **%squitgame**: Stop the currently running game.\n", prefix))

	return infoString.String()
}

// Change the art style of the cards if valid, and return a status message in response.
func setCardsStyle(guildID string, style string) string {
	msg := "No change was made."
	state := GetServerState(guildID)
	newStyle := strings.ToLower(style)
	if newStyle == "normal" {
		state.cardsStyle = KenneyLarge
		msg = "Changed cards to normal style."
	} else if newStyle == "pixel" {
		state.cardsStyle = KenneyPixel
		msg = "Changed cards to pixel style."
	}
	return msg
}

// Toggle whether the Joker cards should be in the deck of cards or not, and return a status message in response.
func toggleJokerCards(guildID string, toggle bool) string {
	msg := "No change was made."
	state := GetServerState(guildID)

	if state.GameType() != NoGame {
		return gameInProgressWarning()
	}

	if toggle && state.includeJokers {
		return "There are already Joker cards in the deck."
	} else if !toggle && !state.includeJokers {
		return "There are no Joker cards in the deck."
	}

	state.includeJokers = toggle
	state.deck = playingcards.NewDeck(state.includeJokers)

	if toggle {
		msg = "Added Joker cards and reset the deck."
	} else {
		msg = "Removed Joker cards and reset the deck."
	}
	return msg
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

	// Set up slash commands
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := dg.ApplicationCommandCreate(AppID, "", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

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

	if *RemoveCommands {
		log.Println("Removing commands...")
		for _, v := range registeredCommands {
			err := dg.ApplicationCommandDelete(dg.State.User.ID, *GuildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}
}

// GetServerState looks for the given server and returns it if it exists, or creates a new entry first
func GetServerState(guildID string) *ServerState {
	state, exists := serverStates[guildID]
	if !exists {
		// Add the server to the list of servers
		state = NewServerState(guildID)
		serverStates[guildID] = state
		// Initialize the server's deck of cards
		state.deck = playingcards.NewDeck(state.includeJokers)
	}
	return state
}

// GetCardPath is specific to how the card images are named
func GetCardPath(card playingcards.Card, style int) string {
	suit := strings.ToLower(card.Suit().String())
	valueString := ""
	path := ""

	if card.NumberAsString() == "Joker" {
		if style == KenneyLarge {
			path = fmt.Sprintf("card_images/kenney_cards_large/cardJoker%s.png", card.Color())
		} else if style == KenneyPixel {
			path = fmt.Sprintf("card_images/kenney_cards_pixel/card_joker_%s.png", strings.ToLower(card.Color()))
		}
		return path
	}

	switch card.Value() {
	case 1:
		valueString = "A"
	case 2, 3, 4, 5, 6, 7, 8, 9, 10:
		if style == KenneyPixel {
			// Kenney's pixel cards are 0-padded
			valueString = fmt.Sprintf("%02d", card.Value())
		} else if style == KenneyLarge {
			valueString = fmt.Sprintf("%d", card.Value())
		}

	case 11:
		valueString = "J"
	case 12:
		valueString = "Q"
	case 13:
		valueString = "K"
	default:
		return ""
	}
	if style == KenneyLarge {
		// Kenney's normal cards are of the format "card<Suit>XX" in camelCase
		path = fmt.Sprintf("card_images/kenney_cards_large/card%s%s.png", strings.ToUpper(suit[0:1])+suit[1:], valueString)
	} else if style == KenneyPixel {
		// Kenney's pixel cards are of the format "card_<suit>_XX"
		path = fmt.Sprintf("card_images/kenney_cards_pixel/card_%s_%s.png", suit, valueString)
	}

	return path
}

// GetCardURL returns the full url to the image for the given card
func GetCardURL(card playingcards.Card, style int) string {
	cardPath := GetCardPath(card, style)
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
		message := &discordgo.MessageEmbed{
			Color:       0x607d8b,
			Title:       "Playing Cards Bot Info",
			Description: getInfoText(),
		}
		s.ChannelMessageSendEmbed(m.ChannelID, message)
	}

	if command == "set_style_normal" {
		msg := setCardsStyle(m.GuildID, "normal")
		s.ChannelMessageSend(m.ChannelID, msg)
	}
	if command == "set_style_pixel" {
		msg := setCardsStyle(m.GuildID, "pixel")
		s.ChannelMessageSend(m.ChannelID, msg)
	}

	if command == "include_jokers" {
		msg := toggleJokerCards(m.GuildID, true)
		s.ChannelMessageSend(m.ChannelID, msg)
	}
	if command == "remove_jokers" {
		msg := toggleJokerCards(m.GuildID, false)
		s.ChannelMessageSend(m.ChannelID, msg)
	}

	// TODO: Relies on message reactions. Switch to using buttons instead.
	if command == "high_or_low" {
		state := GetServerState(m.GuildID)
		if state.GameType() != NoGame {
			s.ChannelMessageSend(m.ChannelID, gameInProgressWarning())
			return
		}
		message := &discordgo.MessageEmbed{
			Color:       0x3dbb6b,
			Title:       "High or Low",
			Description: "Guess whether the next card will be higher or lower.\nReact with 🎲 to join.\nOnly your first reaction in each round will be counted, so choose carefully!",
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Game starting in 7 seconds...",
			},
		}
		messageObj, err := s.ChannelMessageSendEmbed(m.ChannelID, message)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error when trying to start the game.")
			return
		}
		s.MessageReactionAdd(m.ChannelID, messageObj.ID, "\xf0\x9f\x8e\xb2")
		state.game.lastMessageID = messageObj.ID
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
			cardURL := GetCardURL(cardDrawn, state.cardsStyle)
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

		state.deck = playingcards.NewDeck(state.includeJokers)
		s.ChannelMessageSend(m.ChannelID, "Cards have been reset.")
	}
	if command == "quitgame" {
		state := GetServerState(m.GuildID)
		if state.GameType() == NoGame {
			s.ChannelMessageSend(m.ChannelID, "There is no game in progress.")
			return
		}

		state.game.gameType = NoGame
		state.deck = playingcards.NewDeck(state.includeJokers)
		s.ChannelMessageSend(m.ChannelID, "Stopped the game.")
	}
}

func gameInProgressWarning() string {
	return fmt.Sprintf("A game is currently in progress! Enter `%squitgame` to stop the game.", prefix)
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

		if reactionName == "🎲" && state.game.preStartPhase {
			_, ok := state.Players()[m.MessageReaction.UserID]
			// Add new players to the game
			if !ok {
				state.Players()[m.MessageReaction.UserID] = &PlayerState{active: true}
				return
			}
			return
		}

		guess := NoGuess
		if reactionName == "⬆️" {
			guess = High
		} else if reactionName == "⬇️" {
			guess = Low
		} else {
			// Ignore all other reactions
			return
		}
		// Update the player's state based on their guess
		playerState, ok := state.Players()[m.MessageReaction.UserID]
		if ok {
			// Check if the player is still in the game and has not guessed this round yet
			if playerState.Active() && playerState.choice == NoGuess {
				playerState.choice = guess
				state.Players()[m.MessageReaction.UserID] = playerState
			}
		}
	}
}

// HighOrLowGame starts a new game of High or Low for the given Discord server in the channel the bot responded to.
func HighOrLowGame(state *ServerState, s *discordgo.Session, channelID string) {
	state.game.gameType = HighOrLow
	state.deck = playingcards.NewDeck(false) // High or Low does not use Joker cards
	state.deck.Shuffle()
	state.game.channelID = channelID
	state.game.preStartPhase = true
	time.Sleep(7 * time.Second)
	state.game.preStartPhase = false

	// Check if any players have joined and the game did not exit
	if len(state.Players()) == 0 || state.game.gameType != HighOrLow {
		if len(state.Players()) == 0 && state.game.gameType == HighOrLow {
			s.ChannelMessageSend(channelID, "Nobody joined!")
		}
		resetState(state)
		return
	}

	// Set up the game state
	state.game.channelID = channelID
	cardDrawn := state.deck.DrawCard()
	numPlayers := len(state.Players())
	numRounds := 0

	// Game loop
	for {
		if state.game.gameType != HighOrLow {
			resetState(state)
			return
		}
		// Show the current card, add up/down reactions, then wait 5 seconds
		cardURL := GetCardURL(cardDrawn, state.cardsStyle)
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
			resetState(state)
			s.ChannelMessageSend(channelID, "Error found while running the game. Exiting...")
			return
		}
		s.MessageReactionAdd(channelID, messageObj.ID, "\xe2\xac\x86\xef\xb8\x8f")
		s.MessageReactionAdd(channelID, messageObj.ID, "\xe2\xac\x87\xef\xb8\x8f")
		state.game.lastMessageID = messageObj.ID

		time.Sleep(5 * time.Second)

		if state.game.gameType != HighOrLow {
			// The bot quit the game
			resetState(state)
			return
		}

		// Check all players who have reacted, remove wrong responses
		lastCardValue := cardDrawn.Value()
		cardDrawn = state.deck.DrawCard()
		correctGuess := NoGuess // Default is a tie
		guessString := ""
		if cardDrawn.Value() < lastCardValue {
			correctGuess = Low
			guessString = "lower"
		} else if cardDrawn.Value() > lastCardValue {
			correctGuess = High
			guessString = "higher"
		}

		if correctGuess == NoGuess {
			// The new card was neither higher nor lower, nobody is eliminated
			s.ChannelMessageSend(channelID, "Draw! Nobody was eliminated.")
			for _, playerState := range state.Players() {
				// Make sure to reset the players' choices
				if playerState.Active() {
					playerState.choice = NoGuess
				}
			}
		} else {
			eliminatedPlayers := []string{}
			// Iterate through all active players, removing those who made the wrong guess
			for player, playerState := range state.Players() {
				if playerState.Active() && playerState.choice != correctGuess {
					playerState.active = false
					eliminatedPlayers = append(eliminatedPlayers, player)
				}
				// Make sure to reset the player's choice
				playerState.choice = NoGuess
			}
			noMorePlayers := len(eliminatedPlayers) >= numPlayers
			// List the players eliminated this round
			var eliminatedMessage strings.Builder
			eliminatedMessage.WriteString(fmt.Sprintf("%s. The next card was %s!\n", cardDrawn.String(), guessString))
			if len(eliminatedPlayers) == 0 {
				eliminatedMessage.WriteString("No players eliminated.")
			} else {
				eliminatedMessage.WriteString("Players eliminated this round: ")
				for _, player := range eliminatedPlayers {
					// If these players were the last ones eliminated, revert their active status (making them winners)
					if noMorePlayers {
						state.Players()[player].active = true
					}
					member, err := s.GuildMember(state.id, player)
					if err != nil {
						continue
					}
					eliminatedMessage.WriteString(fmt.Sprintf("%s ", member.Mention()))
				}
			}
			s.ChannelMessageSend(channelID, eliminatedMessage.String())

			numPlayers -= len(eliminatedPlayers)
		}

		if numPlayers <= 0 {
			break
		}

		numRounds++

		if state.deck.Size() == 0 {
			// Ran out of cards, end the game
			s.ChannelMessageSend(channelID, "No more cards left!")
			break
		}
	}

	// Print the last card of the game
	cardURL := GetCardURL(cardDrawn, state.cardsStyle)
	message := &discordgo.MessageEmbed{
		Color: 0x3dbb6b,
		Title: fmt.Sprintf("Last card drawn: %s", cardDrawn.String()),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%d cards remained.", state.deck.Size()),
		},
		Image: &discordgo.MessageEmbedImage{
			URL: cardURL,
		},
	}
	s.ChannelMessageSendEmbed(channelID, message)

	// List the winners
	var winnersMessage strings.Builder
	roundString := "rounds"
	if numRounds == 1 {
		roundString = "round"
	}
	winnersMessage.WriteString(fmt.Sprintf("Game end! Congrats to the following players who lasted the most rounds! (%d %s)\n", numRounds, roundString))
	for player, playerState := range state.Players() {
		if playerState.Active() {
			member, err := s.GuildMember(state.id, player)
			if err != nil {
				continue
			}
			winnersMessage.WriteString(fmt.Sprintf("%s ", member.Mention()))
		}
	}
	s.ChannelMessageSend(channelID, winnersMessage.String())

	// Reset game state
	resetState(state)
}

func resetState(state *ServerState) {
	state.game.gameType = NoGame
	state.game.channelID = ""
	state.game.lastMessageID = ""
	state.players = make(map[string]*PlayerState)
	state.deck = playingcards.NewDeck(state.includeJokers)
}
