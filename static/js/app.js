// Yandex Music PWA JavaScript Application

class YandexMusicApp {
    constructor() {
        this.currentTrack = null;
        this.searchResults = [];
        this.currentIndex = -1;
        
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
        
        // Register service worker for PWA
        this.registerServiceWorker();
        
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
            this.displaySearchResults();
            
            if (this.searchResults.length > 0) {
                this.showStatus(`Found ${this.searchResults.length} tracks`);
            } else {
                this.showStatus('No tracks found');
            }
        } catch (error) {
            console.error('Search error:', error);
            this.showError('Failed to search tracks. Please try again.');
        }
    }

    displaySearchResults() {
        this.searchResultsContainer.innerHTML = '';
        
        if (this.searchResults.length === 0) {
            this.searchResultsContainer.innerHTML = '<div class="empty-state">No results found. Try a different search.</div>';
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
                    `<img src="${track.coverUrl}" alt="${track.title} cover" class="result-cover">` : 
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
            `<img src="${this.currentTrack.coverUrl}" alt="${this.escapeHtml(this.currentTrack.title)} cover" class="cover-image">` :
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
            
            // Create a temporary link and trigger download
            const link = document.createElement('a');
            link.href = data.url;
            link.download = `${this.currentTrack.title} - ${this.currentTrack.artist}.mp3`;
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            
            this.showStatus(`Downloading: ${this.currentTrack.title}`);
        } catch (error) {
            console.error('Download error:', error);
            this.showError('Failed to download track. Please try again.');
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
}

// Initialize the app when DOM is loaded
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        new YandexMusicApp();
    });
} else {
    new YandexMusicApp();
}
