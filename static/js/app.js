// Yandex Music PWA JavaScript Application

class YandexMusicApp {
    constructor() {
        this.currentTrack = null;
        this.searchResults = [];
        this.currentIndex = -1;
        this.previousSearchResults = null; // Store previous search state
        this.albums = [];
        this.artists = [];
        
        // DOM Elements
        this.searchForm = document.getElementById('search-form');
        this.searchInput = document.getElementById('search-input');
        this.audioPlayer = document.getElementById('audio-player');
        this.searchResultsContainer = document.getElementById('search-results');
        this.currentTrackInfo = document.getElementById('current-track-info');
        this.prevBtn = document.getElementById('prev-btn');
        this.nextBtn = document.getElementById('next-btn');
        this.downloadBtn = document.getElementById('download-btn');
        this.statusMessage = document.getElementById('status-message');
        
        this.init();
    }

    init() {
        // Event Listeners
        this.searchForm.addEventListener('submit', (e) => this.handleSearch(e));
        this.prevBtn.addEventListener('click', () => this.playPrevious());
        this.nextBtn.addEventListener('click', () => this.playNext());
        this.downloadBtn.addEventListener('click', () => this.downloadTrack());
        
        // Audio player events
        this.audioPlayer.addEventListener('ended', () => this.handleTrackEnded());
        this.audioPlayer.addEventListener('error', (e) => this.handleAudioError(e));
        
        // Media key support
        if ('mediaSession' in navigator) {
            navigator.mediaSession.setActionHandler('previoustrack', () => this.playPrevious());
            navigator.mediaSession.setActionHandler('nexttrack', () => this.playNext());
        }
        
        // Register service worker for PWA
        this.registerServiceWorker();
        
        // Handle URL parameters
        this.handleUrlParameters();
        
        console.log('Yandex Music PWA initialized');
    }

    async registerServiceWorker() {
        if ('serviceWorker' in navigator) {
            try {
                const registration = await navigator.serviceWorker.register('/sw.js');
                console.log('Service Worker registered:', registration);
            } catch (error) {
                console.log('Service Worker registration failed:', error);
            }
        }
    }

    async handleUrlParameters() {
        const urlParams = new URLSearchParams(window.location.search);
        const search = urlParams.get('search');
        const albumId = urlParams.get('album_id');
        const autoplay = urlParams.get('autoplay') === '1';
        
        if (search) {
            // Handle search parameter
            this.searchInput.value = search.replace(/\+/g, ' ');
            await this.handleSearch({ preventDefault: () => {} });
            
            if (autoplay && this.searchResults.length > 0) {
                // Wait a bit for results to be displayed
                setTimeout(() => {
                    this.playTrack(0);
                }, 500);
            }
        } else if (albumId) {
            // Handle album_id parameter
            try {
                this.showLoading();
                const response = await fetch(`/api/album-tracks?id=${albumId}&name=Album`);
                if (response.ok) {
                    const data = await response.json();
                    this.searchResults = data.tracks || [];
                    
                    this.searchResultsContainer.innerHTML = '';
                    const albumHeader = document.createElement('h2');
                    albumHeader.textContent = `Album (${this.searchResults.length} tracks)`;
                    albumHeader.className = 'results-header';
                    this.searchResultsContainer.appendChild(albumHeader);
                    
                    this.searchResults.forEach((track, index) => {
                        const resultItem = document.createElement('div');
                        resultItem.className = 'result-item';
                        resultItem.setAttribute('role', 'button');
                        resultItem.setAttribute('tabindex', '0');
                        resultItem.setAttribute('aria-label', `Play ${track.title} by ${track.artist}`);
                        
                        resultItem.innerHTML = `
                            ${track.coverUrl ? 
                                `<img src="${track.coverUrl}" alt="${track.title} cover" class="result-cover" onerror="this.style.display='none'; this.nextElementSibling.style.display='block';">
                                 <div class="result-cover" style="display:none;"></div>` : 
                                '<div class="result-cover"></div>'
                            }
                            <div class="result-info">
                                <div class="result-title">${this.escapeHtml(track.title)}</div>
                                <div class="result-artist">${this.escapeHtml(track.artist)}</div>
                            </div>
                            <div class="result-duration">${this.formatDuration(track.duration)}</div>
                        `;
                        
                        resultItem.addEventListener('click', () => this.playTrack(index));
                        resultItem.addEventListener('keypress', (e) => {
                            if (e.key === 'Enter' || e.key === ' ') {
                                e.preventDefault();
                                this.playTrack(index);
                            }
                        });
                        
                        this.searchResultsContainer.appendChild(resultItem);
                    });
                    
                    if (autoplay && this.searchResults.length > 0) {
                        setTimeout(() => {
                            this.playTrack(0);
                        }, 500);
                    }
                    
                    this.showStatus(`Loaded ${this.searchResults.length} tracks from album`);
                }
            } catch (error) {
                console.error('Album loading error:', error);
                this.showError('Failed to load album. Please try again.');
            }
        }
    }

    async handleSearch(e) {
        e.preventDefault();
        
        const query = this.searchInput.value.trim();
        if (query.length < 3) {
            this.showStatus('Please enter at least 3 characters');
            return;
        }

        this.showLoading();
        
        try {
            const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`);
            if (!response.ok) {
                throw new Error('Search failed');
            }
            
            const data = await response.json();
            this.searchResults = data.tracks || [];
            this.albums = data.albums || [];
            this.artists = data.artists || [];
            
            // Store search state for back navigation
            this.previousSearchResults = {
                tracks: this.searchResults,
                albums: this.albums,
                artists: this.artists,
                data: data
            };
            
            this.displaySearchResults(data);
            
            const totalResults = this.searchResults.length + this.albums.length + this.artists.length;
            if (totalResults > 0) {
                this.showStatus(`Found ${totalResults} results`);
            } else {
                this.showStatus('No results found');
            }
        } catch (error) {
            console.error('Search error:', error);
            this.showError('Failed to search. Please try again.');
        }
    }

    displaySearchResults(data) {
        this.searchResultsContainer.innerHTML = '';
        
        const totalResults = this.searchResults.length + (this.albums?.length || 0) + (this.artists?.length || 0);
        
        if (totalResults === 0) {
            this.searchResultsContainer.innerHTML = '<div class="empty-state">No results found. Try a different search.</div>';
            return;
        }

        // Display spelling correction if applicable
        if (data.misspellCorrected && data.correctedText) {
            const correctionNotice = document.createElement('div');
            correctionNotice.className = 'correction-notice';
            correctionNotice.innerHTML = `Showing results for: <strong>${this.escapeHtml(data.correctedText)}</strong>`;
            correctionNotice.setAttribute('role', 'status');
            this.searchResultsContainer.appendChild(correctionNotice);
        }

        // Display tracks section
        if (this.searchResults.length > 0) {
            const tracksHeader = document.createElement('h2');
            tracksHeader.textContent = `Tracks (${this.searchResults.length})`;
            tracksHeader.className = 'results-header';
            this.searchResultsContainer.appendChild(tracksHeader);

            this.searchResults.forEach((track, index) => {
                const resultItem = document.createElement('div');
                resultItem.className = 'result-item';
                resultItem.setAttribute('role', 'button');
                resultItem.setAttribute('tabindex', '0');
                resultItem.setAttribute('aria-label', `Play ${track.title} by ${track.artist}`);
                
                resultItem.innerHTML = `
                    ${track.coverUrl ? 
                        `<img src="${track.coverUrl}" alt="${track.title} cover" class="result-cover" onerror="this.style.display='none'; this.nextElementSibling.style.display='block';">
                         <div class="result-cover" style="display:none;"></div>` : 
                        '<div class="result-cover"></div>'
                    }
                    <div class="result-info">
                        <div class="result-title">${this.escapeHtml(track.title)}</div>
                        <div class="result-artist">${this.escapeHtml(track.artist)}</div>
                        ${track.album ? `<div class="result-album-link">
                            <a href="#" data-album-id="${track.albumId}" data-album-name="${this.escapeHtml(track.album)}" 
                               onclick="event.stopPropagation(); return false;">
                                ${this.escapeHtml(track.album)}
                            </a>
                        </div>` : ''}
                    </div>
                    <div class="result-duration">${this.formatDuration(track.duration)}</div>
                `;
                
                resultItem.addEventListener('click', () => this.playTrack(index));
                resultItem.addEventListener('keypress', (e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        this.playTrack(index);
                    }
                });

                // Handle album link clicks
                const albumLink = resultItem.querySelector('.result-album-link a');
                if (albumLink) {
                    albumLink.addEventListener('click', (e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        const albumId = e.target.dataset.albumId;
                        const albumName = e.target.dataset.albumName;
                        this.loadAlbumTracks(albumId, albumName);
                    });
                }
                
                this.searchResultsContainer.appendChild(resultItem);
            });
        }

        // Display albums section
        if (this.albums && this.albums.length > 0) {
            const albumsHeader = document.createElement('h2');
            albumsHeader.textContent = `Albums (${this.albums.length})`;
            albumsHeader.className = 'results-header';
            this.searchResultsContainer.appendChild(albumsHeader);

            this.albums.forEach((album) => {
                console.log('[App] Creating album item:', {
                    title: album.title,
                    coverUrl: album.coverUrl,
                    hasCover: !!album.coverUrl
                });
                
                const albumItem = document.createElement('div');
                albumItem.className = 'result-item album-item';
                albumItem.setAttribute('role', 'button');
                albumItem.setAttribute('tabindex', '0');
                albumItem.setAttribute('aria-label', `View album ${album.title} by ${album.artist}`);
                
                albumItem.innerHTML = `
                    ${album.coverUrl ? 
                        `<img src="${album.coverUrl}" alt="${album.title} cover" class="result-cover" onerror="this.style.display='none'; this.nextElementSibling.style.display='block'; console.error('[App] Image load error:', '${album.coverUrl}');">
                         <div class="result-cover" style="display:none;"></div>` : 
                        '<div class="result-cover"></div>'
                    }
                    <div class="result-info">
                        <div class="result-title">${this.escapeHtml(album.title)}</div>
                        <div class="result-artist">${this.escapeHtml(album.artist)}</div>
                        ${album.year ? `<div class="result-meta">${album.year}</div>` : ''}
                        ${album.trackCount ? `<div class="result-meta">${album.trackCount} tracks</div>` : ''}
                    </div>
                `;
                
                albumItem.addEventListener('click', () => this.loadAlbumTracks(album.id, album.title));
                albumItem.addEventListener('keypress', (e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        this.loadAlbumTracks(album.id, album.title);
                    }
                });
                
                this.searchResultsContainer.appendChild(albumItem);
            });
        }

        // Display artists section
        if (this.artists && this.artists.length > 0) {
            const artistsHeader = document.createElement('h2');
            artistsHeader.textContent = `Artists (${this.artists.length})`;
            artistsHeader.className = 'results-header';
            this.searchResultsContainer.appendChild(artistsHeader);

            this.artists.forEach((artist) => {
                const artistItem = document.createElement('div');
                artistItem.className = 'result-item artist-item';
                artistItem.setAttribute('role', 'button');
                artistItem.setAttribute('tabindex', '0');
                artistItem.setAttribute('aria-label', `View artist ${artist.name}`);
                
                artistItem.innerHTML = `
                    ${artist.coverUrl ? 
                        `<img src="${artist.coverUrl}" alt="${artist.name}" class="result-cover" onerror="this.style.display='none'; this.nextElementSibling.style.display='block';">
                         <div class="result-cover" style="display:none;"></div>` : 
                        '<div class="result-cover"></div>'
                    }
                    <div class="result-info">
                        <div class="result-title">${this.escapeHtml(artist.name)}</div>
                    </div>
                `;
                
                artistItem.addEventListener('click', () => this.loadArtistTracks(artist.id, artist.name));
                artistItem.addEventListener('keypress', (e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        this.loadArtistTracks(artist.id, artist.name);
                    }
                });
                
                this.searchResultsContainer.appendChild(artistItem);
            });
        }
    }

    async playTrack(index) {
        if (index < 0 || index >= this.searchResults.length) {
            this.showError('Invalid track index');
            return;
        }

        this.currentIndex = index;
        this.currentTrack = this.searchResults[index];
        
        this.showStatus('Loading track...');
        
        try {
            const response = await fetch(`/api/download-url?id=${this.currentTrack.id}`);
            if (!response.ok) {
                throw new Error('Failed to get track URL');
            }
            
            const data = await response.json();
            
            // Set audio source
            this.audioPlayer.src = data.url;
            this.audioPlayer.load();
            
            // Play the track
            try {
                await this.audioPlayer.play();
                this.updateCurrentTrackDisplay();
                this.updateControls();
                this.showStatus(`Now playing: ${this.currentTrack.title} by ${this.currentTrack.artist}`);
            } catch (playError) {
                console.error('Play error:', playError);
                this.showError('Failed to play track. Please try again.');
            }
        } catch (error) {
            console.error('Track loading error:', error);
            this.showError('Failed to load track. Please try again.');
        }
    }

    updateCurrentTrackDisplay() {
        if (!this.currentTrack) {
            this.currentTrackInfo.innerHTML = `
                <div class="cover-placeholder" aria-hidden="true">
                    <svg width="80" height="80" viewBox="0 0 80 80" fill="currentColor">
                        <path d="M40 0C17.9 0 0 17.9 0 40s17.9 40 40 40 40-17.9 40-40S62.1 0 40 0zm0 72c-17.6 0-32-14.4-32-32S22.4 8 40 8s32 14.4 32 32-14.4 32-32 32z"/>
                        <circle cx="40" cy="40" r="8"/>
                    </svg>
                </div>
                <div class="track-details">
                    <p class="no-track-message">No track loaded. Search for music to get started.</p>
                </div>
            `;
            return;
        }

        const coverHtml = this.currentTrack.coverUrl ?
            `<img src="${this.currentTrack.coverUrl}" alt="${this.escapeHtml(this.currentTrack.title)} cover" class="cover-image" onerror="this.style.display='none'; this.nextElementSibling.style.display='block';">
             <div class="cover-placeholder" aria-hidden="true" style="display:none;">
                <svg width="80" height="80" viewBox="0 0 80 80" fill="currentColor">
                    <path d="M40 0C17.9 0 0 17.9 0 40s17.9 40 40 40 40-17.9 40-40S62.1 0 40 0zm0 72c-17.6 0-32-14.4-32-32S22.4 8 40 8s32 14.4 32 32-14.4 32-32 32z"/>
                    <circle cx="40" cy="40" r="8"/>
                </svg>
            </div>` :
            `<div class="cover-placeholder" aria-hidden="true">
                <svg width="80" height="80" viewBox="0 0 80 80" fill="currentColor">
                    <path d="M40 0C17.9 0 0 17.9 0 40s17.9 40 40 40 40-17.9 40-40S62.1 0 40 0zm0 72c-17.6 0-32-14.4-32-32S22.4 8 40 8s32 14.4 32 32-14.4 32-32 32z"/>
                    <circle cx="40" cy="40" r="8"/>
                </svg>
            </div>`;

        this.currentTrackInfo.innerHTML = `
            ${coverHtml}
            <div class="track-details">
                <div class="track-title">${this.escapeHtml(this.currentTrack.title)}</div>
                <div class="track-artist">${this.escapeHtml(this.currentTrack.artist)}</div>
            </div>
        `;
    }

    updateControls() {
        // Enable/disable previous button
        this.prevBtn.disabled = this.currentIndex <= 0;
        
        // Enable/disable next button
        this.nextBtn.disabled = this.currentIndex >= this.searchResults.length - 1;
        
        // Enable download button if track is available
        this.downloadBtn.disabled = !this.currentTrack;
    }

    playPrevious() {
        if (this.currentIndex > 0) {
            this.playTrack(this.currentIndex - 1);
        }
    }

    playNext() {
        if (this.currentIndex < this.searchResults.length - 1) {
            this.playTrack(this.currentIndex + 1);
        }
    }

    handleTrackEnded() {
        // Auto-play next track if available
        if (this.currentIndex < this.searchResults.length - 1) {
            this.playNext();
        } else {
            this.showStatus('Playlist ended');
        }
    }

    handleAudioError(e) {
        console.error('Audio error:', e);
        this.showError('Audio playback error occurred');
    }

    async downloadTrack() {
        if (!this.currentTrack) {
            this.showError('No track selected');
            return;
        }

        try {
            const response = await fetch(`/api/download-url?id=${this.currentTrack.id}`);
            if (!response.ok) {
                throw new Error('Failed to get download URL');
            }
            
            const data = await response.json();
            
            // Trigger actual download using fetch and blob
            this.showStatus(`Downloading: ${this.currentTrack.title}`);
            
            const downloadResponse = await fetch(data.url);
            const blob = await downloadResponse.blob();
            
            // Create a blob URL and trigger download
            const blobUrl = URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.href = blobUrl;
            link.download = `${this.currentTrack.title} - ${this.currentTrack.artist}.mp3`;
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            
            // Clean up the blob URL
            setTimeout(() => URL.revokeObjectURL(blobUrl), 100);
            
            this.showStatus(`Downloaded: ${this.currentTrack.title}`);
        } catch (error) {
            console.error('Download error:', error);
            this.showError('Failed to download track. Please try again.');
        }
    }

    async loadAlbumTracks(albumId, albumName) {
        this.showLoading();
        this.showStatus(`Loading album: ${albumName}`);
        
        console.log('[App] Loading album tracks:', { albumId, albumName });
        
        try {
            const response = await fetch(`/api/album-tracks?id=${albumId}&name=${encodeURIComponent(albumName)}`);
            console.log('[App] Album tracks response:', {
                status: response.status,
                ok: response.ok,
                type: response.type
            });
            
            if (!response.ok) {
                throw new Error('Failed to load album tracks');
            }
            
            const data = await response.json();
            console.log('[App] Album tracks data:', data);
            
            this.searchResults = data.tracks || [];
            
            this.searchResultsContainer.innerHTML = '';
            
            // Add back button
            const backBtn = document.createElement('button');
            backBtn.className = 'back-button';
            backBtn.textContent = '← Back to search results';
            backBtn.onclick = () => {
                this.restorePreviousSearch();
            };
            this.searchResultsContainer.appendChild(backBtn);
            
            // Add album header
            const albumHeader = document.createElement('h2');
            albumHeader.textContent = `Album: ${albumName}`;
            albumHeader.className = 'results-header';
            this.searchResultsContainer.appendChild(albumHeader);
            
            if (this.searchResults.length === 0) {
                this.searchResultsContainer.innerHTML += '<div class="empty-state">No tracks found in this album.</div>';
                return;
            }
            
            this.searchResults.forEach((track, index) => {
                const resultItem = document.createElement('div');
                resultItem.className = 'result-item';
                resultItem.setAttribute('role', 'button');
                resultItem.setAttribute('tabindex', '0');
                resultItem.setAttribute('aria-label', `Play ${track.title} by ${track.artist}`);
                
                console.log('[App] Creating track item:', {
                    title: track.title,
                    coverUrl: track.coverUrl,
                    hasCover: !!track.coverUrl
                });
                
                resultItem.innerHTML = `
                    ${track.coverUrl ? 
                        `<img src="${track.coverUrl}" alt="${track.title} cover" class="result-cover" onerror="this.style.display='none'; this.nextElementSibling.style.display='block'; console.error('[App] Image load error:', '${track.coverUrl}');">
                         <div class="result-cover" style="display:none;"></div>` : 
                        '<div class="result-cover"></div>'
                    }
                    <div class="result-info">
                        <div class="result-title">${this.escapeHtml(track.title)}</div>
                        <div class="result-artist">${this.escapeHtml(track.artist)}</div>
                    </div>
                    <div class="result-duration">${this.formatDuration(track.duration)}</div>
                `;
                
                resultItem.addEventListener('click', () => this.playTrack(index));
                resultItem.addEventListener('keypress', (e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        this.playTrack(index);
                    }
                });
                
                this.searchResultsContainer.appendChild(resultItem);
            });
            
            this.showStatus(`Loaded ${this.searchResults.length} tracks from album: ${albumName}`);
        } catch (error) {
            console.error('Album loading error:', error);
            this.showError('Failed to load album tracks. Please try again.');
        }
    }

    async loadArtistTracks(artistId, artistName) {
        this.showLoading();
        this.showStatus(`Loading tracks by: ${artistName}`);
        
        try {
            const response = await fetch(`/api/artist-tracks?id=${artistId}&name=${encodeURIComponent(artistName)}`);
            if (!response.ok) {
                throw new Error('Failed to load artist tracks');
            }
            
            const data = await response.json();
            this.searchResults = data.tracks || [];
            
            this.searchResultsContainer.innerHTML = '';
            
            // Add back button
            const backBtn = document.createElement('button');
            backBtn.className = 'back-button';
            backBtn.textContent = '← Back to search results';
            backBtn.onclick = () => {
                this.restorePreviousSearch();
            };
            this.searchResultsContainer.appendChild(backBtn);
            
            // Add artist header
            const artistHeader = document.createElement('h2');
            artistHeader.textContent = `Artist: ${artistName}`;
            artistHeader.className = 'results-header';
            this.searchResultsContainer.appendChild(artistHeader);
            
            if (this.searchResults.length === 0) {
                this.searchResultsContainer.innerHTML += '<div class="empty-state">No tracks found for this artist.</div>';
                return;
            }
            
            this.searchResults.forEach((track, index) => {
                const resultItem = document.createElement('div');
                resultItem.className = 'result-item';
                resultItem.setAttribute('role', 'button');
                resultItem.setAttribute('tabindex', '0');
                resultItem.setAttribute('aria-label', `Play ${track.title} by ${track.artist}`);
                
                resultItem.innerHTML = `
                    ${track.coverUrl ? 
                        `<img src="${track.coverUrl}" alt="${track.title} cover" class="result-cover" onerror="this.style.display='none'; this.nextElementSibling.style.display='block';">
                         <div class="result-cover" style="display:none;"></div>` : 
                        '<div class="result-cover"></div>'
                    }
                    <div class="result-info">
                        <div class="result-title">${this.escapeHtml(track.title)}</div>
                        <div class="result-artist">${this.escapeHtml(track.artist)}</div>
                        ${track.album ? `<div class="result-meta">${this.escapeHtml(track.album)}</div>` : ''}
                    </div>
                    <div class="result-duration">${this.formatDuration(track.duration)}</div>
                `;
                
                resultItem.addEventListener('click', () => this.playTrack(index));
                resultItem.addEventListener('keypress', (e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        this.playTrack(index);
                    }
                });
                
                this.searchResultsContainer.appendChild(resultItem);
            });
            
            this.showStatus(`Loaded ${this.searchResults.length} tracks by: ${artistName}`);
        } catch (error) {
            console.error('Artist loading error:', error);
            this.showError('Failed to load artist tracks. Please try again.');
        }
    }

    showLoading() {
        this.searchResultsContainer.innerHTML = '<div class="loading">Searching...</div>';
    }

    showError(message) {
        const errorDiv = document.createElement('div');
        errorDiv.className = 'error';
        errorDiv.textContent = message;
        errorDiv.setAttribute('role', 'alert');
        
        this.searchResultsContainer.insertBefore(errorDiv, this.searchResultsContainer.firstChild);
        this.showStatus(message);
        
        setTimeout(() => {
            errorDiv.remove();
        }, 5000);
    }

    showStatus(message) {
        this.statusMessage.textContent = message;
        console.log('Status:', message);
    }

    formatDuration(ms) {
        const seconds = Math.floor(ms / 1000);
        const minutes = Math.floor(seconds / 60);
        const remainingSeconds = seconds % 60;
        return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    restorePreviousSearch() {
        if (!this.previousSearchResults) {
            this.showError('No previous search results to restore');
            return;
        }
        
        // Restore search state
        this.searchResults = this.previousSearchResults.tracks;
        this.albums = this.previousSearchResults.albums;
        this.artists = this.previousSearchResults.artists;
        
        // Re-display the previous search results
        this.displaySearchResults(this.previousSearchResults.data);
        this.showStatus('Returned to previous search results');
    }
}

// Initialize the app when DOM is loaded
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        new YandexMusicApp();
    });
} else {
    new YandexMusicApp();
}
