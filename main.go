package main

import (
	"fmt"
	"github.com/svntax/PlayingCardsBot/playingcards"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())

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
}
