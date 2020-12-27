package main

import (
	"fmt"
	"github.com/svntax/PlayingCardsBot/playingcards"
)

func main() {
	testCard := playingcards.NewCard(1, "clubs")
	testCard2 := playingcards.NewCard(5, "diamonds")
	testCard3 := playingcards.NewCard(10, "hearts")
	testCard4 := playingcards.NewCard(11, "Spades")
	testCard5 := playingcards.NewCard(12, "DIAMONDS")
	testCard6 := playingcards.NewCard(13, "Hearts")
	fmt.Println(testCard)
	fmt.Println(testCard2)
	fmt.Println(testCard3)
	fmt.Println(testCard4)
	fmt.Println(testCard5)
	fmt.Println(testCard6)
}
