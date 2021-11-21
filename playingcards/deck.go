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

// NewDeckWithoutJokers creates a new deck of cards with no Joker cards
func NewDeckWithoutJokers() Deck {
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

// NewDeckWithJokers creates a new deck of cards with a red and black Joker included
func NewDeckWithJokers() Deck {
	deck := make([]Card, 54)
	i := 0
	for suit := CLUBS; suit <= SPADES; suit++ {
		for n := 1; n <= 13; n++ {
			card := NewCard(n, suit)
			deck[i] = card
			i++
		}
	}
	redJoker := NewCard(-1, RED_JOKER)
	deck[i] = redJoker
	i++
	blackJoker := NewCard(-1, BLACK_JOKER)
	deck[i] = blackJoker
	return Deck{cards: deck}
}

// NewDeck creates a new deck of cards
func NewDeck(includeJokers bool) Deck {
	if includeJokers {
		return NewDeckWithJokers()
	} else {
		return NewDeckWithoutJokers()
	}
}

// Size returns the number of cards remaining in this deck
func (d Deck) Size() int {
	return len(d.cards)
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
