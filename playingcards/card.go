package playingcards

import (
	"fmt"
	"strconv"
	"strings"
)

// Suit is clubs, diamonds, hearts, or spades
type Suit int

// Constants for suit types
const (
	CLUBS Suit = iota
	DIAMONDS
	HEARTS
	SPADES
	RED_JOKER
	BLACK_JOKER
)

func (s Suit) String() string {
	switch s {
	case CLUBS:
		return "Clubs"
	case DIAMONDS:
		return "Diamonds"
	case HEARTS:
		return "Hearts"
	case SPADES:
		return "Spades"
	case RED_JOKER:
		return "Red Joker"
	case BLACK_JOKER:
		return "Black Joker"
	default:
		panic("Invalid suit value")
	}
}

// NumberAsString returns the proper string representation of a playing card's value
func (c Card) NumberAsString() string {
	switch c.number {
	case 1:
		return "Ace"
	case 2, 3, 4, 5, 6, 7, 8, 9, 10:
		return strconv.Itoa(c.number)
	case 11:
		return "Jack"
	case 12:
		return "Queen"
	case 13:
		return "King"
	case -1:
		return "Joker"
	default:
		return "Invalid value"
	}
}

// Card is a standard playing card
type Card struct {
	number int
	suit   Suit
}

// SuitStringToInt returns the int equivalent of the given string
func SuitStringToInt(suit string) Suit {
	switch strings.ToUpper(suit) {
	case "CLUBS":
		return CLUBS
	case "DIAMONDS":
		return DIAMONDS
	case "HEARTS":
		return HEARTS
	case "SPADES":
		return SPADES
	case "RED_JOKER":
		return RED_JOKER
	case "BLACK_JOKER":
		return BLACK_JOKER
	default:
		panic("A card's suit must be Clubs, Diamonds, Hearts, or Spades.")
	}
}

// NewCard creates a new playing card
func NewCard(num int, s Suit) Card {
	c := Card{number: num, suit: s}
	return c
}

// Color returns the card's color
func (c Card) Color() string {
	switch c.Suit() {
	case CLUBS, SPADES, BLACK_JOKER:
		return "Black"
	case HEARTS, DIAMONDS, RED_JOKER:
		return "Red"
	default:
		panic("Invalid suit value")
	}
}

// Suit returns the card's suit
func (c Card) Suit() Suit {
	return c.suit
}

// Value returns the card's value
func (c Card) Value() int {
	return c.number
}

func (c Card) String() string {
	if c.suit == RED_JOKER {
		return "Red Joker"
	}
	if c.suit == BLACK_JOKER {
		return "Black Joker"
	}
	return fmt.Sprintf("%s of %s", c.NumberAsString(), c.Suit())
}
