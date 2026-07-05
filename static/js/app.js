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
        this.downloadAlbumBtn = document.getElementById('download-album-btn');
        this.statusMessage = document.getElementById('status-message');

        this.init();
    }

    init() {
        this.searchForm.addEventListener('submit', (e) => this.handleSearch(e));
        this.prevBtn.addEventListener('click', () => this.playPrevious());
        this.nextBtn.addEventListener('click', () => this.playNext());
        this.downloadBtn.addEventListener('click', () => this.downloadTrack());
        this.downloadAlbumBtn.addEventListener('click', () => {
            if (this.currentAlbumInfo) {
                this.downloadAlbumZip(this.currentAlbumInfo.id, this.currentAlbumInfo.name);
            }
        });

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
    // Supported params (any of these can be put in a link to this site):
    //   ?track_id=123                     — open (and auto-play) a single track
    //   ?album_id=123[&album_name=...]    — open an album
    //   ?artist_id=123[&artist_name=...]  — open an artist's tracks
    //   ?search=query[&autoplay=1]        — run a search, optionally auto-play first result
    //   &download=1                       — additionally trigger a download once loaded
    //     (works together with track_id or album_id)
    async handleUrlParameters() {
        const p = new URLSearchParams(window.location.search);
        const search     = p.get('search');
        const albumId    = p.get('album_id');
        const albumName  = p.get('album_name') || '';
        const artistId   = p.get('artist_id');
        const artistName = p.get('artist_name') || '';
        const trackId    = p.get('track_id');
        const autoplay   = p.get('autoplay') === '1';
        const download   = p.get('download') === '1';

        if (trackId) {
            // loadTrackById already plays the track once it's loaded.
            await this.loadTrackById(trackId, albumId || null);
            if (download && this.currentTrack) await this.downloadTrack();
        } else if (search) {
            this.searchInput.value = search.replace(/\+/g, ' ');
            await this.handleSearch({ preventDefault: () => {} });
            if (autoplay && this.searchResults.length > 0) await this.playTrack(0);
        } else if (albumId) {
            // If album_name wasn't given, fetch with a blank name and let the
            // API return whatever the album is actually called.
            await this.loadAlbumTracks(albumId, albumName);
            if (autoplay && this.searchResults.length > 0) await this.playTrack(0);
            if (download && this.currentAlbumInfo) {
                this.downloadAlbumZip(albumId, this.currentAlbumInfo.name);
            }
        } else if (artistId) {
            await this.loadArtistTracks(artistId, artistName);
            if (autoplay && this.searchResults.length > 0) await this.playTrack(0);
        }
    }

    // ── Shareable links ──────────────────────────────────────────────────────
    // Builds an absolute URL back to this same page (respects base path/proxy
    // prefix automatically, since it's built from the current location).
    buildShareUrl(params) {
        const url = new URL(window.location.origin + window.location.pathname);
        url.search = '';
        Object.entries(params).forEach(([key, value]) => {
            if (value !== undefined && value !== null && value !== '') {
                url.searchParams.set(key, value);
            }
        });
        return url.toString();
    }

    async copyShareLink(url, what) {
        try {
            await navigator.clipboard.writeText(url);
            this.showStatus(`Link to ${what} copied to clipboard`);
        } catch (err) {
            // Clipboard API unavailable (e.g. insecure context) — fall back to a prompt.
            window.prompt(`Copy this link to ${what}:`, url);
        }
    }

    // ── Yandex Music URL parser ────────────────────────────────────────────────
    // Supported formats:
    //   https://music.yandex.ru/album/12345
    //   https://music.yandex.ru/album/12345/track/67890
    //   https://music.yandex.com/album/12345
    //   https://music.yandex.com/album/12345/track/67890
    // Returns { type: 'album'|'track', albumId, trackId } or null
    parseYandexUrl(input) {
        const s = input.trim();
        // Must look like a URL
        if (!s.includes('music.yandex.')) return null;
        try {
            // Allow pasting bare URLs without scheme
            const url = new URL(s.startsWith('http') ? s : 'https://' + s);
            if (!url.hostname.includes('music.yandex.')) return null;
            // pathname: /album/12345 or /album/12345/track/67890
            const m = url.pathname.match(/\/album\/(\d+)(?:\/track\/(\d+))?/);
            if (!m) return null;
            return {
                type:    m[2] ? 'track' : 'album',
                albumId: m[1],
                trackId: m[2] || null,
            };
        } catch {
            return null;
        }
    }

    // ── Search ───────────────────────────────────────────────────────────────
    async handleSearch(e) {
        e.preventDefault();
        const query = this.searchInput.value.trim();
        if (query.length < 3) { this.showStatus('Please enter at least 3 characters'); return; }

        // ── Yandex Music URL? Handle directly ────────────────────────────────
        const parsed = this.parseYandexUrl(query);
        if (parsed) {
            if (parsed.type === 'track') {
                await this.loadTrackById(parsed.trackId, parsed.albumId);
            } else {
                await this.loadAlbumTracks(parsed.albumId, '');
            }
            return;
        }

        this.showLoading();
        try {
            const resp = await fetch(`api/search?q=${encodeURIComponent(query)}`);
            if (!resp.ok) throw new Error('Search failed');
            const data = await resp.json();

            this.searchResults = data.tracks  || [];
            this.albums        = data.albums  || [];
            this.artists       = data.artists || [];
            this.currentAlbumInfo = null;
            this.updateAlbumDownloadBtn();

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
            <button type="button" class="icon-btn share-btn" aria-label="Copy link to ${this.escapeHtml(track.title)}">${this.shareIconSvg()}</button>
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

        const shareBtn = item.querySelector('.share-btn');
        if (shareBtn) {
            shareBtn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                const url = this.buildShareUrl({ track_id: track.id, album_id: track.albumId || '' });
                this.copyShareLink(url, `track "${track.title}"`);
            });
        }

        this.searchResultsContainer.appendChild(item);
    }

    // Small reusable link/share SVG icon
    shareIconSvg() {
        return `<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M3.9 12c0-1.71 1.39-3.1 3.1-3.1h4V7H7c-2.76 0-5 2.24-5 5s2.24 5 5 5h4v-1.9H7c-1.71 0-3.1-1.39-3.1-3.1zM8 13h8v-2H8v2zm9-6h-4v1.9h4c1.71 0 3.1 1.39 3.1 3.1s-1.39 3.1-3.1 3.1h-4V17h4c2.76 0 5-2.24 5-5s-2.24-5-5-5z"/>
        </svg>`;
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
            <button type="button" class="icon-btn share-btn" aria-label="Copy link to album ${this.escapeHtml(album.title)}">${this.shareIconSvg()}</button>
        `;

        item.addEventListener('click', () => this.loadAlbumTracks(album.id, album.title));
        item.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); this.loadAlbumTracks(album.id, album.title); }
        });

        const shareBtn = item.querySelector('.share-btn');
        if (shareBtn) {
            shareBtn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                const url = this.buildShareUrl({ album_id: album.id, album_name: album.title || '' });
                this.copyShareLink(url, `album "${album.title}"`);
            });
        }

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
            <button type="button" class="icon-btn share-btn" aria-label="Copy link to artist ${this.escapeHtml(artist.name)}">${this.shareIconSvg()}</button>
        `;

        item.addEventListener('click', () => this.loadArtistTracks(artist.id, artist.name));
        item.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); this.loadArtistTracks(artist.id, artist.name); }
        });

        const shareBtn = item.querySelector('.share-btn');
        if (shareBtn) {
            shareBtn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                const url = this.buildShareUrl({ artist_id: artist.id, artist_name: artist.name || '' });
                this.copyShareLink(url, `artist "${artist.name}"`);
            });
        }

        this.searchResultsContainer.appendChild(item);
    }

    // ── Load single track by ID (from URL) ────────────────────────────────────
    async loadTrackById(trackId, albumId) {
        this.showLoading();
        this.showStatus('Loading track…');
        try {
            const resp = await fetch(`api/track-info?id=${trackId}`);
            if (!resp.ok) {
                const err = await resp.json().catch(() => ({}));
                throw new Error(err.error || 'Failed to load track');
            }
            const track = await resp.json();

            // Put it in a one-item playlist so prev/next logic still works
            this.searchResults = [track];
            this.currentAlbumInfo = null;
            this.updateAlbumDownloadBtn();

            // Show track in results panel with option to open its album
            this.searchResultsContainer.innerHTML = '';
            const header = document.createElement('h2');
            header.className = 'results-header';
            header.textContent = 'Track from URL';
            this.searchResultsContainer.appendChild(header);

            if (track.albumId) {
                const albumBtn = document.createElement('button');
                albumBtn.className = 'back-button';
                albumBtn.setAttribute('aria-label', `Open album ${track.album || 'containing this track'}`);
                albumBtn.textContent = `📀 Open album: ${track.album || 'View album'}`;
                albumBtn.onclick = () => this.loadAlbumTracks(track.albumId, track.album || '');
                this.searchResultsContainer.appendChild(albumBtn);
            }

            this.appendTrackItem(track, 0);
            await this.playTrack(0);
        } catch (err) {
            console.error(err);
            this.showError(`Could not load track: ${err.message}`);
        }
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
            this.updateAlbumDownloadBtn();

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

            const shareAlbumBtn = document.createElement('button');
            shareAlbumBtn.className = 'control-btn share-btn-inline';
            shareAlbumBtn.setAttribute('aria-label', `Copy link to album ${albumName || 'Album'}`);
            shareAlbumBtn.innerHTML = `${this.shareIconSvg()} Copy link`;
            shareAlbumBtn.onclick = () => {
                const url = this.buildShareUrl({ album_id: albumId, album_name: albumName || '' });
                this.copyShareLink(url, `album "${albumName || 'Album'}"`);
            };
            headerRow.appendChild(shareAlbumBtn);

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
            this.updateAlbumDownloadBtn();

            this.searchResultsContainer.innerHTML = '';

            if (this.previousSearchResults) {
                const backBtn = document.createElement('button');
                backBtn.className = 'back-button';
                backBtn.textContent = '← Back to search results';
                backBtn.onclick = () => this.restorePreviousSearch();
                this.searchResultsContainer.appendChild(backBtn);
            }

            const artistHeaderRow = document.createElement('div');
            artistHeaderRow.style.cssText = 'display:flex;align-items:center;gap:1rem;flex-wrap:wrap;margin-bottom:0.5rem;';
            const artistH = document.createElement('h2');
            artistH.className = 'results-header';
            artistH.style.margin = '0';
            artistH.textContent = `Artist: ${artistName}`;
            artistHeaderRow.appendChild(artistH);

            const shareArtistBtn = document.createElement('button');
            shareArtistBtn.className = 'control-btn share-btn-inline';
            shareArtistBtn.setAttribute('aria-label', `Copy link to artist ${artistName}`);
            shareArtistBtn.innerHTML = `${this.shareIconSvg()} Copy link`;
            shareArtistBtn.onclick = () => {
                const url = this.buildShareUrl({ artist_id: artistId, artist_name: artistName || '' });
                this.copyShareLink(url, `artist "${artistName}"`);
            };
            artistHeaderRow.appendChild(shareArtistBtn);

            this.searchResultsContainer.appendChild(artistHeaderRow);

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

    updateAlbumDownloadBtn() {
        if (this.currentAlbumInfo) {
            this.downloadAlbumBtn.hidden = false;
            this.downloadAlbumBtn.disabled = false;
            this.downloadAlbumBtn.setAttribute('aria-label',
                `Download album "${this.currentAlbumInfo.name}" as ZIP`);
        } else {
            this.downloadAlbumBtn.hidden = true;
            this.downloadAlbumBtn.disabled = true;
            this.downloadAlbumBtn.setAttribute('aria-label', 'Download current album as ZIP');
        }
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
