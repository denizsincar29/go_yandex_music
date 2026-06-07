// Yandex Music PWA — rewritten with bug fixes + album zip + auto-play next

class YandexMusicApp {
    constructor() {
        this.currentTrack = null;
        this.searchResults = [];   // current track list (may be album/artist view)
        this.currentIndex = -1;
        this.previousSearchResults = null; // search state for back navigation
        this.albums = [];
        this.artists = [];
        this.currentAlbumInfo = null; // { id, name } for zip download

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
        this.searchForm.addEventListener('submit', (e) => this.handleSearch(e));
        this.prevBtn.addEventListener('click', () => this.playPrevious());
        this.nextBtn.addEventListener('click', () => this.playNext());
        this.downloadBtn.addEventListener('click', () => this.downloadTrack());

        // Auto-play next when track ends
        this.audioPlayer.addEventListener('ended', () => this.handleTrackEnded());
        // On audio error — skip to next track automatically
        this.audioPlayer.addEventListener('error', (e) => this.handleAudioError(e));

        // Media session keys
        if ('mediaSession' in navigator) {
            navigator.mediaSession.setActionHandler('previoustrack', () => this.playPrevious());
            navigator.mediaSession.setActionHandler('nexttrack', () => this.playNext());
        }

        this.registerServiceWorker();
        this.handleUrlParameters();
    }

    async registerServiceWorker() {
        if ('serviceWorker' in navigator) {
            try {
                await navigator.serviceWorker.register('sw.js');
            } catch (e) {
                console.warn('SW registration failed:', e);
            }
        }
    }

    // ── Cover URL helper ─────────────────────────────────────────────────────
    // Yandex returns URIs with %% placeholder for size, e.g. "…/%%"
    // We replace with a concrete size to get a real image.
    fixCoverUrl(rawUrl) {
        if (!rawUrl) return '';
        let url = rawUrl;
        // Replace %% size placeholder
        url = url.replace(/%%/g, '200x200');
        // Ensure https:// prefix
        if (!url.startsWith('http')) url = 'https://' + url;
        return url;
    }

    // ── URL parameter handling ───────────────────────────────────────────────
    async handleUrlParameters() {
        const p = new URLSearchParams(window.location.search);
        const search  = p.get('search');
        const albumId = p.get('album_id');
        const autoplay = p.get('autoplay') === '1';

        if (search) {
            this.searchInput.value = search.replace(/\+/g, ' ');
            await this.handleSearch({ preventDefault: () => {} });
            if (autoplay && this.searchResults.length > 0) await this.playTrack(0);
        } else if (albumId) {
            // We don't know the name yet — fetch with a blank name then let
            // the API return whatever the album is actually called.
            await this.loadAlbumTracks(albumId, '');
            if (autoplay && this.searchResults.length > 0) await this.playTrack(0);
        }
    }

    // ── Search ───────────────────────────────────────────────────────────────
    async handleSearch(e) {
        e.preventDefault();
        const query = this.searchInput.value.trim();
        if (query.length < 3) { this.showStatus('Please enter at least 3 characters'); return; }

        this.showLoading();
        try {
            const resp = await fetch(`api/search?q=${encodeURIComponent(query)}`);
            if (!resp.ok) throw new Error('Search failed');
            const data = await resp.json();

            this.searchResults = data.tracks  || [];
            this.albums        = data.albums  || [];
            this.artists       = data.artists || [];
            this.currentAlbumInfo = null;

            this.previousSearchResults = {
                tracks: this.searchResults,
                albums: this.albums,
                artists: this.artists,
                data
            };

            this.displaySearchResults(data);
            const total = this.searchResults.length + this.albums.length + this.artists.length;
            this.showStatus(total > 0 ? `Found ${total} results` : 'No results found');
        } catch (err) {
            console.error(err);
            this.showError('Search failed. Please try again.');
        }
    }

    // ── Display search results ────────────────────────────────────────────────
    displaySearchResults(data) {
        this.searchResultsContainer.innerHTML = '';
        const total = this.searchResults.length + (this.albums?.length || 0) + (this.artists?.length || 0);

        if (total === 0) {
            this.searchResultsContainer.innerHTML = '<div class="empty-state">No results found.</div>';
            return;
        }

        if (data.misspellCorrected && data.correctedText) {
            const notice = document.createElement('div');
            notice.className = 'correction-notice';
            notice.setAttribute('role', 'status');
            notice.innerHTML = `Showing results for: <strong>${this.escapeHtml(data.correctedText)}</strong>`;
            this.searchResultsContainer.appendChild(notice);
        }

        // Tracks
        if (this.searchResults.length > 0) {
            this.appendHeader(`Tracks (${this.searchResults.length})`);
            this.searchResults.forEach((track, i) => this.appendTrackItem(track, i));
        }

        // Albums
        if (this.albums?.length > 0) {
            this.appendHeader(`Albums (${this.albums.length})`);
            this.albums.forEach(album => this.appendAlbumItem(album));
        }

        // Artists
        if (this.artists?.length > 0) {
            this.appendHeader(`Artists (${this.artists.length})`);
            this.artists.forEach(artist => this.appendArtistItem(artist));
        }
    }

    appendHeader(text) {
        const h = document.createElement('h2');
        h.className = 'results-header';
        h.textContent = text;
        this.searchResultsContainer.appendChild(h);
    }

    appendTrackItem(track, index) {
        const item = document.createElement('div');
        item.className = 'result-item';
        item.id = `track-item-${index}`;
        item.setAttribute('role', 'button');
        item.setAttribute('tabindex', '0');
        item.setAttribute('aria-label', `Play ${track.title} by ${track.artist}`);

        const coverUrl = this.fixCoverUrl(track.coverUrl);
        item.innerHTML = `
            ${coverUrl
                ? `<img src="${coverUrl}" alt="" class="result-cover" onerror="this.style.display='none'">`
                : '<div class="result-cover" aria-hidden="true"></div>'}
            <div class="result-info">
                <div class="result-title">${this.escapeHtml(track.title)}</div>
                <div class="result-artist">${this.escapeHtml(track.artist)}</div>
                ${track.album
                    ? `<div class="result-album-link">
                         <a href="#" data-album-id="${track.albumId}" data-album-name="${this.escapeHtml(track.album)}">
                             ${this.escapeHtml(track.album)}
                         </a>
                       </div>`
                    : ''}
            </div>
            <div class="result-duration">${this.formatDuration(track.duration)}</div>
        `;

        item.addEventListener('click', () => this.playTrack(index));
        item.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); this.playTrack(index); }
        });

        const albumLink = item.querySelector('.result-album-link a');
        if (albumLink) {
            albumLink.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                this.loadAlbumTracks(albumLink.dataset.albumId, albumLink.dataset.albumName);
            });
        }

        this.searchResultsContainer.appendChild(item);
    }

    appendAlbumItem(album) {
        const item = document.createElement('div');
        item.className = 'result-item album-item';
        item.setAttribute('role', 'button');
        item.setAttribute('tabindex', '0');
        item.setAttribute('aria-label', `View album ${album.title} by ${album.artist}`);

        const coverUrl = this.fixCoverUrl(album.coverUrl);
        item.innerHTML = `
            ${coverUrl
                ? `<img src="${coverUrl}" alt="" class="result-cover" onerror="this.style.display='none'">`
                : '<div class="result-cover" aria-hidden="true"></div>'}
            <div class="result-info">
                <div class="result-title">${this.escapeHtml(album.title)}</div>
                <div class="result-artist">${this.escapeHtml(album.artist)}</div>
                ${album.year       ? `<div class="result-meta">${album.year}</div>` : ''}
                ${album.trackCount ? `<div class="result-meta">${album.trackCount} tracks</div>` : ''}
            </div>
        `;

        item.addEventListener('click', () => this.loadAlbumTracks(album.id, album.title));
        item.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); this.loadAlbumTracks(album.id, album.title); }
        });
        this.searchResultsContainer.appendChild(item);
    }

    appendArtistItem(artist) {
        const item = document.createElement('div');
        item.className = 'result-item artist-item';
        item.setAttribute('role', 'button');
        item.setAttribute('tabindex', '0');
        item.setAttribute('aria-label', `View artist ${artist.name}`);

        const coverUrl = this.fixCoverUrl(artist.coverUrl);
        item.innerHTML = `
            ${coverUrl
                ? `<img src="${coverUrl}" alt="" class="result-cover" onerror="this.style.display='none'">`
                : '<div class="result-cover" aria-hidden="true"></div>'}
            <div class="result-info">
                <div class="result-title">${this.escapeHtml(artist.name)}</div>
            </div>
        `;

        item.addEventListener('click', () => this.loadArtistTracks(artist.id, artist.name));
        item.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); this.loadArtistTracks(artist.id, artist.name); }
        });
        this.searchResultsContainer.appendChild(item);
    }

    // ── Load album tracks ─────────────────────────────────────────────────────
    async loadAlbumTracks(albumId, albumName) {
        this.showLoading();
        this.showStatus(`Loading album…`);

        try {
            const nameParam = albumName ? encodeURIComponent(albumName) : 'Album';
            const resp = await fetch(`api/album-tracks?id=${albumId}&name=${nameParam}`);
            if (!resp.ok) throw new Error('Failed to load album tracks');
            const data = await resp.json();

            this.searchResults = data.tracks || [];
            this.currentAlbumInfo = { id: albumId, name: albumName || 'Album' };

            this.searchResultsContainer.innerHTML = '';

            // Back button
            if (this.previousSearchResults) {
                const backBtn = document.createElement('button');
                backBtn.className = 'back-button';
                backBtn.textContent = '← Back to search results';
                backBtn.onclick = () => this.restorePreviousSearch();
                this.searchResultsContainer.appendChild(backBtn);
            }

            // Header + Zip download button
            const headerRow = document.createElement('div');
            headerRow.style.cssText = 'display:flex;align-items:center;gap:1rem;flex-wrap:wrap;margin-bottom:0.5rem;';
            const h = document.createElement('h2');
            h.className = 'results-header';
            h.style.margin = '0';
            h.textContent = `Album: ${albumName || 'Album'}`;
            headerRow.appendChild(h);

            const zipBtn = document.createElement('button');
            zipBtn.className = 'control-btn zip-btn';
            zipBtn.setAttribute('aria-label', `Download all tracks from ${albumName || 'album'} as ZIP`);
            zipBtn.innerHTML = `
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                    <path d="M20 6h-2.18c.07-.44.18-.88.18-1.36 0-2.57-2.1-4.64-4.69-4.64-1.53 0-2.9.72-3.82 1.82C8.76 1.1 7.5.5 6 .5 3.52.5 1.5 2.5 1.5 5c0 .48.11.92.18 1.36H0v14c0 1.1.9 2 2 2h20c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm-5-4c1.47 0 2.69 1.2 2.69 2.64 0 .48-.13.94-.35 1.36H13v-.68l1.5-1.5V5h-1v-.68L12 2.82c.59-.5 1.36-.82 2.18-.82H15zM6 2.5c1.24 0 2.28.82 2.66 1.94L7.5 5.58V6H6.5v-.68L5 3.86C5.37 3.04 6.14 2.5 6 2.5zm14 17.5H4V8h16v12z"/>
                </svg>
                Download all as ZIP
            `;
            zipBtn.onclick = () => this.downloadAlbumZip(albumId, albumName || 'Album');
            headerRow.appendChild(zipBtn);
            this.searchResultsContainer.appendChild(headerRow);

            if (this.searchResults.length === 0) {
                const empty = document.createElement('div');
                empty.className = 'empty-state';
                empty.textContent = 'No tracks found in this album.';
                this.searchResultsContainer.appendChild(empty);
                return;
            }

            this.searchResults.forEach((track, i) => this.appendTrackItem(track, i));
            this.showStatus(`Loaded ${this.searchResults.length} tracks from album: ${albumName || 'Album'}`);
        } catch (err) {
            console.error(err);
            this.showError('Failed to load album tracks. Please try again.');
        }
    }

    // ── Load artist tracks ────────────────────────────────────────────────────
    async loadArtistTracks(artistId, artistName) {
        this.showLoading();
        this.showStatus(`Loading tracks by: ${artistName}`);

        try {
            const resp = await fetch(`api/artist-tracks?id=${artistId}&name=${encodeURIComponent(artistName)}`);
            if (!resp.ok) throw new Error('Failed to load artist tracks');
            const data = await resp.json();

            this.searchResults = data.tracks || [];
            this.currentAlbumInfo = null;

            this.searchResultsContainer.innerHTML = '';

            if (this.previousSearchResults) {
                const backBtn = document.createElement('button');
                backBtn.className = 'back-button';
                backBtn.textContent = '← Back to search results';
                backBtn.onclick = () => this.restorePreviousSearch();
                this.searchResultsContainer.appendChild(backBtn);
            }

            this.appendHeader(`Artist: ${artistName}`);

            if (this.searchResults.length === 0) {
                const empty = document.createElement('div');
                empty.className = 'empty-state';
                empty.textContent = 'No tracks found for this artist.';
                this.searchResultsContainer.appendChild(empty);
                return;
            }

            this.searchResults.forEach((track, i) => this.appendTrackItem(track, i));
            this.showStatus(`Loaded ${this.searchResults.length} tracks by: ${artistName}`);
        } catch (err) {
            console.error(err);
            this.showError('Failed to load artist tracks. Please try again.');
        }
    }

    // ── Playback ──────────────────────────────────────────────────────────────
    async playTrack(index) {
        if (index < 0 || index >= this.searchResults.length) return;

        this.currentIndex = index;
        this.currentTrack = this.searchResults[index];

        // Highlight active track in list
        document.querySelectorAll('.result-item.active').forEach(el => el.classList.remove('active'));
        const activeEl = document.getElementById(`track-item-${index}`);
        if (activeEl) {
            activeEl.classList.add('active');
            activeEl.scrollIntoView({ block: 'nearest' });
        }

        this.showStatus(`Loading: ${this.currentTrack.title}…`);

        try {
            const resp = await fetch(`api/download-url?id=${this.currentTrack.id}`);
            if (!resp.ok) throw new Error('Failed to get track URL');
            const data = await resp.json();

            this.audioPlayer.src = data.url;
            this.audioPlayer.load();
            await this.audioPlayer.play();

            this.updateCurrentTrackDisplay();
            this.updateControls();
            this.updateMediaSession();
            this.showStatus(`Now playing: ${this.currentTrack.title} by ${this.currentTrack.artist}`);
        } catch (err) {
            console.error('Track load error:', err);
            this.showError(`Failed to load "${this.currentTrack.title}" — skipping…`);
            // Auto-skip on error after a short pause
            setTimeout(() => this.playNext(), 1500);
        }
    }

    playPrevious() {
        if (this.currentIndex > 0) this.playTrack(this.currentIndex - 1);
    }

    playNext() {
        if (this.currentIndex < this.searchResults.length - 1) {
            this.playTrack(this.currentIndex + 1);
        } else {
            this.showStatus('Playlist ended');
        }
    }

    handleTrackEnded() {
        this.playNext();
    }

    handleAudioError(e) {
        // Only auto-skip if we actually have a current track loaded
        if (this.currentTrack) {
            console.error('Audio error, skipping track:', e);
            this.showStatus(`Playback error on "${this.currentTrack.title}" — skipping…`);
            setTimeout(() => this.playNext(), 1500);
        }
    }

    // ── Controls & display ────────────────────────────────────────────────────
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
                </div>`;
            return;
        }

        const coverUrl = this.fixCoverUrl(this.currentTrack.coverUrl);
        this.currentTrackInfo.innerHTML = `
            ${coverUrl
                ? `<img src="${coverUrl}" alt="" class="cover-image" onerror="this.style.display='none'">`
                : `<div class="cover-placeholder" aria-hidden="true">
                       <svg width="80" height="80" viewBox="0 0 80 80" fill="currentColor">
                           <path d="M40 0C17.9 0 0 17.9 0 40s17.9 40 40 40 40-17.9 40-40S62.1 0 40 0zm0 72c-17.6 0-32-14.4-32-32S22.4 8 40 8s32 14.4 32 32-14.4 32-32 32z"/>
                           <circle cx="40" cy="40" r="8"/>
                       </svg>
                   </div>`}
            <div class="track-details">
                <div class="track-title">${this.escapeHtml(this.currentTrack.title)}</div>
                <div class="track-artist">${this.escapeHtml(this.currentTrack.artist)}</div>
            </div>`;
    }

    updateControls() {
        this.prevBtn.disabled = this.currentIndex <= 0;
        this.nextBtn.disabled = this.currentIndex >= this.searchResults.length - 1;
        this.downloadBtn.disabled = !this.currentTrack;
    }

    updateMediaSession() {
        if (!('mediaSession' in navigator) || !this.currentTrack) return;
        const coverUrl = this.fixCoverUrl(this.currentTrack.coverUrl);
        navigator.mediaSession.metadata = new MediaMetadata({
            title:   this.currentTrack.title,
            artist:  this.currentTrack.artist,
            album:   this.currentTrack.album || '',
            artwork: coverUrl ? [{ src: coverUrl }] : []
        });
    }

    // ── Single track download ─────────────────────────────────────────────────
    async downloadTrack() {
        if (!this.currentTrack) { this.showError('No track selected'); return; }
        try {
            const resp = await fetch(`api/download-url?id=${this.currentTrack.id}`);
            if (!resp.ok) throw new Error('Failed to get download URL');
            const { url } = await resp.json();

            this.showStatus(`Downloading: ${this.currentTrack.title}…`);
            const blob = await (await fetch(url)).blob();
            const blobUrl = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = blobUrl;
            a.download = `${this.currentTrack.title} - ${this.currentTrack.artist}.mp3`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            setTimeout(() => URL.revokeObjectURL(blobUrl), 100);
            this.showStatus(`Downloaded: ${this.currentTrack.title}`);
        } catch (err) {
            console.error(err);
            this.showError('Download failed. Please try again.');
        }
    }

    // ── Album ZIP download ────────────────────────────────────────────────────
    downloadAlbumZip(albumId, albumName) {
        const url = `api/album-zip?id=${albumId}&name=${encodeURIComponent(albumName)}`;
        this.showStatus(`Preparing ZIP for "${albumName}"… This may take a while.`);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${albumName}.zip`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
    }

    // ── Restore previous search ───────────────────────────────────────────────
    restorePreviousSearch() {
        if (!this.previousSearchResults) { this.showError('No previous search to restore'); return; }
        this.searchResults    = this.previousSearchResults.tracks;
        this.albums           = this.previousSearchResults.albums;
        this.artists          = this.previousSearchResults.artists;
        this.currentAlbumInfo = null;
        this.displaySearchResults(this.previousSearchResults.data);
        // Re-highlight the currently playing track if it was in the search
        if (this.currentIndex >= 0) {
            const el = document.getElementById(`track-item-${this.currentIndex}`);
            if (el) el.classList.add('active');
        }
        this.showStatus('Returned to search results');
    }

    // ── Utilities ─────────────────────────────────────────────────────────────
    showLoading() {
        this.searchResultsContainer.innerHTML = '<div class="loading" aria-live="polite">Loading…</div>';
    }

    showError(message) {
        const div = document.createElement('div');
        div.className = 'error';
        div.setAttribute('role', 'alert');
        div.textContent = message;
        this.searchResultsContainer.insertBefore(div, this.searchResultsContainer.firstChild);
        this.showStatus(message);
        setTimeout(() => div.remove(), 5000);
    }

    showStatus(message) {
        this.statusMessage.textContent = message;
    }

    formatDuration(ms) {
        const s = Math.floor(ms / 1000);
        return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
    }

    escapeHtml(text) {
        const d = document.createElement('div');
        d.textContent = text || '';
        return d.innerHTML;
    }
}

// Boot
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => new YandexMusicApp());
} else {
    new YandexMusicApp();
}
