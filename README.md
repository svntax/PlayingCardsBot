# Playing Cards Bot
A Discord bot that gives servers a standard 52-card deck of cards for users to play with. Made using [DiscordGo](https://github.com/bwmarrin/discordgo).

## Commands

| Command | Description |
| --- | --- |
| /info, $pcb info | Displays bot info and a list of all commands. |
| /draw, $pcb draw | Draws a card from the current deck. |
| /shuffle, $pcb shuffle | Shuffles the current deck of cards. |
| /reset-cards, $pcb reset_cards | Replaces the current deck with a brand new, ordered deck of 52 cards. |
| /set-style | Change the style of the cards. Options are "normal" and "pixel". |
| (Old) $pcb set_style_normal | Changes the art style of the cards to normal. |
| (Old) $pcb set_style_pixel | Changes the art style of the cards to pixel art. |
| /include-jokers | Add or remove the red & black Joker cards from the deck. |
| (Old) $pcb include_jokers | Add the red and black Joker cards to the deck. |
| (Old) $pcb remove_jokers | Remove the red and black Joker cards from the deck. |
| $pcb high_or_low | Starts a game of High or Low. |
| /quit-game, $pcb quitgame | Stops any currently running game. |

The list of commands can also be found on the live website (https://playing-cards-bot-rvpup.ondigitalocean.app/).

## Setup
Go to the live website (https://playing-cards-bot-rvpup.ondigitalocean.app/), and click on "Add to Discord" to be redirected to Discord, log in, then choose a server to add the bot to. Note that you'll need your own Discord server or have access to a server where you have `Manage Server` permissions.

## How To Deploy

To host your own instance of this bot, you'll first need to set up a new application and bot on Discord here: https://discordapp.com/developers/applications/.

After setting up the bot application, click on the Deploy to DigitalOcean Button below.

[![Deploy to DO](https://www.deploytodo.com/do-btn-blue.svg)](https://cloud.digitalocean.com/apps/new?repo=https://github.com/svntax/PlayingCardsBot/tree/main)

The app uses two environment variables:

`BOT_TOKEN` is the secret token you can get from your bot application created on Discord. Make sure this token is kept secret.

`HOST_URL` is the url of where the bot will be hosted (e.g., `https://yourcustomdomain.tld`). If you have a custom domain, enter it in full here.

To add the bot to Discord servers, you need to generate an OAuth2 link by going to your bot application in the Discord Developer Portal, clicking OAuth2, "bot" for the scope, and checking the following permissions:
- View Channels
- Send Messages
- Embed Links
- Read Message History
- Add Reactions

The generated link should have the following format: `https://discord.com/api/oauth2/authorize?client_id=<your bot's client id>&permissions=85056&scope=bot`

Alternatively, you can just copy-paste the above link and replace `<your bot's client id>` with the client ID found in General Information.

Once the bot is deployed, feel free to make changes to the frontend, such as replacing the "Add to Discord" button's link with your own bot's.

## Development
The backend consists of `main.go` and the `playingcards` module for cards functionality (e.g., drawing, shuffling cards).

The frontend, found in `/public/`, uses plain HTML and CSS and is served by the backend.

The `/card_images/` directory contains the playing cards images the bot will use, so you can freely add your own sets of images for the bot to use, but you'll have to update the code to support using more image sets. The two current image sets are by [Kenney](https://www.kenney.nl/), which you can find here:
- https://www.kenney.nl/assets/boardgame-pack
- https://www.kenney.nl/assets/playing-cards-pack

The bot can be hosted locally by running `go run main.go -t=<your bot token> -app=<your bot application ID>`, but the card images will not display since they won't be reachable within Discord. Without a `HOST_URL`, the bot will be running on `http://localhost:8080` by default.