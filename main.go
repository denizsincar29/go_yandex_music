package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"

	"github.com/denizsincar29/goerror"
	"github.com/joho/godotenv"
)

func main() {
	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	e := goerror.NewError(l)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	err := godotenv.Load()
	e.Must(err)
	token := os.Getenv("YA_MUSIC_TOKEN")
	uid, err := strconv.Atoi(os.Getenv("YA_MUSIC_ID"))
	e.Must(err)
	player, err := NewPlayer(ctx, uid, token)
	e.Must(err)
	defer player.Close()
	stdin := bufio.NewScanner(os.Stdin)
	query := ""
	fmt.Println("Enter your search query (or 'exit' to quit):")
	stdin.Scan()
	query = stdin.Text()
	if query == "exit" {
		fmt.Println("Exiting...")
		return
	}
	_, err = player.SearchTracks(query)
	e.Must(err)
	player.PlayFirst()
	fmt.Println("Playing first track...")
	title, artist := player.GetCurrentTrack()
	fmt.Printf("Now playing: %s - %s\n", title, artist)
	fmt.Println("Press 'n' for next track, 'p' for previous track, or 'exit' to quit:")
	for {
		stdin.Scan()
		// check for ctrl+c
		select {
		case <-ctx.Done():
			fmt.Println("Exiting...")
			return
		default:
			// continue
		}
		input := stdin.Text()
		switch input {
		case "n":
			err := player.PlayNext()
			if err != nil {
				fmt.Println("Error playing next track:", err)
			} else {
				title, artist := player.GetCurrentTrack()
				fmt.Printf("Now playing: %s - %s\n", title, artist)
			}
		case "p":
			err := player.PlayPrevious()
			if err != nil {
				fmt.Println("Error playing previous track:", err)
			} else {
				title, artist := player.GetCurrentTrack()
				fmt.Printf("Now playing: %s - %s\n", title, artist)
			}
		case "pp":
			if !player.IsPaused() {
				player.Pause()
				fmt.Println("Paused")
			} else {
				player.Resume()
				fmt.Println("Playing")
			}
		case "dl", "download":
			err := player.DownloadTrack("downloads/")
			e.Check(err)
		case "exit", "":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid input. Please enter 'n', 'p', or 'exit'.")
		}
	}

}
