package main

import "pkg.botr.me/yamusic"

type YandexMusicTrack struct {
	ID             int      "json:\"id\""
	DurationMs     int      "json:\"durationMs\""
	Available      bool     "json:\"available\""
	AvailableAsRbt bool     "json:\"availableAsRbt\""
	Explicit       bool     "json:\"explicit\""
	StorageDir     string   "json:\"storageDir\""
	Title          string   "json:\"title\""
	Version        string   "json:\"version,omitempty\""
	Regions        []string "json:\"regions\""
	Albums         []struct {
		ID                  int              "json:\"id\""
		StorageDir          string           "json:\"storageDir\""
		OriginalReleaseYear int              "json:\"originalReleaseYear\""
		Year                int              "json:\"year\""
		Title               string           "json:\"title\""
		Artists             []yamusic.Artist "json:\"artists\""
		CoverURI            string           "json:\"coverUri\""
		TrackCount          int              "json:\"trackCount\""
		Genre               string           "json:\"genre\""
		Available           bool             "json:\"available\""
		TrackPosition       struct {
			Volume int "json:\"volume\""
			Index  int "json:\"index\""
		} "json:\"trackPosition\""
	} "json:\"albums\""
	Artists []yamusic.Artist "json:\"artists\""
}
