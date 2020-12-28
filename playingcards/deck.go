package playingcards

import (
	"math/rand"
)

// EmptyCard is a fake card to return on invalid function calls that return a playing card
var EmptyCard Card = NewCard(-1, CLUBS)

// Deck is a standard 52-card list of playing cards
type Deck struct {
	cards []Card
}

// NewDeck creates a new deck of cards
func NewDeck() Deck {
	deck := make([]Card, 52)
	i := 0
	for suit := CLUBS; suit <= SPADES; suit++ {
		for n := 1; n <= 13; n++ {
			card := NewCard(n, suit)
			deck[i] = card
			i++
		}
	}
	return Deck{cards: deck}
}

// DrawCard removes the top card from the deck and returns it
func (d *Deck) DrawCard() Card {
	if len(d.cards) > 0 {
		topCard := d.cards[len(d.cards)-1]
		d.cards = d.cards[:len(d.cards)-1]
		return topCard
	}
	d.cards = nil
	return EmptyCard
}

// Shuffle randomizes the order of the remaining cards in the deck
func (d *Deck) Shuffle() {
	rand.Shuffle(len(d.cards), func(i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
}
