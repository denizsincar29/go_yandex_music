package main

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/denizsincar29/goerror"
	"github.com/joho/godotenv"
)

//go:embed version.txt
var version string

func main() {
	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	e := goerror.NewError(l)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	fmt.Println("Yandex Music player")
	if version != "" {
		fmt.Println("Version:", version)
	}
	err := update(ctx, version)
	e.Debug(err)
	err = godotenv.Load()
	e.Must(err)
	token := os.Getenv("YA_MUSIC_TOKEN")
	uid, err := strconv.Atoi(os.Getenv("YA_MUSIC_ID"))
	e.Must(err)
	player, err := NewPlayer(ctx, uid, token)
	e.Must(err)
	defer player.Close()
	printWelcome()
	stdin := bufio.NewScanner(os.Stdin)
	for {
		stdin.Scan()
		// check for ctrl+c
		//playing_done := player.finished
		select {
		case <-ctx.Done():
			fmt.Println("Exiting...")
			return
			//case <-playing_done:
			//	fmt.Println("Playing done")
		//	player.PlayNext()
		default:
			// continue
		}
		input := stdin.Text()
		cmd := strings.SplitN(input, " ", 2)
		switch cmd[0] {
		case "h", "help":
			printHelp()
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
			title, artist := player.GetCurrentTrack()
			fmt.Printf("Downloading: %s - %s\n", title, artist)
			err := player.DownloadTrack("downloads/")
			e.Check(err)
			fmt.Println("Download complete.")
		case "exit", "":
			fmt.Println("Exiting...")
			return
		default:
			// if the input is <3 letters, skip it
			if len(input) < 3 {
				fmt.Println("The input is too short.")
				continue
			}
			_, err := player.SearchTracks(input)
			if err != nil {
				fmt.Println("Error searching tracks:", err)
				continue
			}
			player.PlayFirst()
			title, artist := player.GetCurrentTrack()
			fmt.Printf("Now playing: %s - %s\n", title, artist)
		}
	}

}
