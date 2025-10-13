package main

import "fmt"

var help = `
Commands:
  s, search <term> - Search for a track and play it
  n, next         - Play the next track in the queue
  p, previous     - Play the previous track in the queue
  pp, pause       - Pause or resume playback
  dl, download    - Download the current track
  exit           - Exit the program
`

func printHelp() {
	fmt.Println(help)
}

// printWelcome prints a welcome message and the help information.
func printWelcome() {
	fmt.Println("Welcome to the yandex Music Player!")
	fmt.Println("Type 'help' for a list of commands.")
}
