package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"pkg.botr.me/yamusic"
)

// struct for yandex music player

type MusicPlayer struct {
	player  *StreamPlayer
	client  *yamusic.Client
	Results []YandexMusicTrack
	ctx     context.Context
	idx     int
}

// NewPlayer creates a new MusicPlayer instance
func NewPlayer(ctx context.Context, uid int, token string) (*MusicPlayer, error) {
	player, err := NewStreamPlayer(nil)
	if err != nil {
		return nil, err
	}
	client := yamusic.NewClient(yamusic.AccessToken(uid, token))
	return &MusicPlayer{player: player, client: client, ctx: ctx, Results: []YandexMusicTrack{}, idx: 0}, nil
}

// SearchTracks searches for tracks using the Yandex Music API
func (m *MusicPlayer) SearchTracks(query string) ([]YandexMusicTrack, error) {
	s, resp, err := m.client.Search().Tracks(m.ctx, query, &yamusic.SearchOptions{Page: 0, NoCorrect: false})
	if err != nil {
		return []YandexMusicTrack{}, err
	}
	if resp.StatusCode != 200 {
		return []YandexMusicTrack{}, fmt.Errorf("error: %s", resp.Status)
	}
	results := s.Result.Tracks.Results
	r := make([]YandexMusicTrack, len(results))
	for i, result := range results {
		r[i] = YandexMusicTrack(result)
	}
	m.Results = r
	m.idx = 0
	return r, nil
}

// PlayTrack plays a track using the Yandex Music API
// receives the track ID as a parameter
func (m *MusicPlayer) PlayTrack(trackID int, resetIndex bool) error {
	url, err := m.client.Tracks().GetDownloadURL(m.ctx, trackID)
	if err != nil {
		return err
	}
	err = m.player.PlayAnotherURL(url)
	if err != nil {
		return err
	}
	if resetIndex {
		m.idx = 0
	}
	return nil
}

// PlayIndex plays a track at the given index in the search results
func (m *MusicPlayer) PlayIndex(index int) error {
	if index < 0 || index >= len(m.Results) {
		return fmt.Errorf("index out of range")
	}
	m.idx = index
	return m.PlayTrack(m.Results[index].ID, false)
}

// PlayNext plays the next track in the search results
func (m *MusicPlayer) PlayNext() error {
	if m.idx+1 >= len(m.Results) {
		return fmt.Errorf("no more tracks")
	}
	m.idx++
	return m.PlayIndex(m.idx)
}

// PlayPrevious plays the previous track in the search results
func (m *MusicPlayer) PlayPrevious() error {
	if m.idx-1 < 0 {
		return fmt.Errorf("no more tracks")
	}
	m.idx--
	return m.PlayIndex(m.idx)
}

// PlayFirst plays the first track in the search results
func (m *MusicPlayer) PlayFirst() error {
	return m.PlayIndex(0)
}

// PlayLast plays the last track in the search results
func (m *MusicPlayer) PlayLast() error {
	return m.PlayIndex(len(m.Results) - 1)
}

// IsPlaying returns true if the player is currently playing a track
func (m *MusicPlayer) IsPlaying() bool {
	return m.player.IsPlaying()
}

// IsPaused returns true if the player is currently paused
func (m *MusicPlayer) IsPaused() bool {
	return m.player.IsPaused()
}

// Pause pauses the current track
func (m *MusicPlayer) Pause() {
	m.player.Pause()
}

// Resume resumes the current track
func (m *MusicPlayer) Resume() {
	m.player.Resume()
}

// Stop stops the current track
func (m *MusicPlayer) Stop() {
	m.player.Stop()
}

// Close closes the player
func (m *MusicPlayer) Close() {
	m.player.Close()
}

// GetCurrentTrack returns the current track info, like title and artist
func (m *MusicPlayer) GetCurrentTrack() (string, string) {
	if m.idx < 0 || m.idx >= len(m.Results) {
		return "", ""
	}
	track := m.Results[m.idx]
	//return track.Title, track.Artists
	artists := ""
	for i, artist := range track.Artists {
		if i == len(track.Artists)-1 {
			artists += artist.Name
		} else {
			artists += artist.Name + ", "
		}
	}
	return track.Title, artists
}

// DownloadTrack downloads the current track
func (m *MusicPlayer) DownloadTrack(dir string) error {
	if m.idx < 0 || m.idx >= len(m.Results) {
		return fmt.Errorf("no track to download")
	}
	track := m.Results[m.idx]
	url, err := m.client.Tracks().GetDownloadURL(m.ctx, track.ID)
	if err != nil {
		return err
	}
	title, artists := m.GetCurrentTrack()
	filename := dir + title + " (" + artists + ").mp3"
	return m.DownloadFile(filename, url)

}

// DownloadFile downloads a file from the given URL and saves it to the given filename
// uses m.client.Do(...)
func (m *MusicPlayer) DownloadFile(filename string, url string) error {
	req, err := m.client.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := m.client.Do(m.ctx, req, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("error: %s", resp.Status)
	}
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil

}
