# Reverse Proxy Setup Guide

This guide explains how to deploy the Yandex Music Player web application behind a reverse proxy at a subpath (e.g., `example.com/music/` instead of the root path).

## Quick Start

### Modern Approach: X-Forwarded-Prefix Header (Recommended)

The application now supports the `X-Forwarded-Prefix` header, which is the modern cloud-native way to handle base paths. Your reverse proxy simply adds this header, and the app automatically adjusts.

**Nginx Example:**
```nginx
location /music/ {
    proxy_pass http://localhost:8080/;
    proxy_set_header X-Forwarded-Prefix /music;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

**Apache Example:**
```apache
<Location /music>
    ProxyPass http://localhost:8080/
    ProxyPassReverse http://localhost:8080/
    RequestHeader set X-Forwarded-Prefix "/music"
    ProxyPreserveHost On
</Location>
```

**Benefits:**
- ✅ No BASE_PATH environment variable needed
- ✅ Dynamic - change paths without restarting the app
- ✅ Works like FastAPI/Uvicorn's proxy headers
- ✅ Enabled by default

**To disable X-Forwarded-Prefix support:**
```bash
USE_PROXY_HEADERS=false ./ya_music_web
```

### Legacy Approach: BASE_PATH Environment Variable

You can still use the static BASE_PATH environment variable if you prefer:

### 1. Set the Base Path Environment Variable

Add `BASE_PATH` to your `.env` file or export it before starting the server:

```bash
# In .env file
BASE_PATH=/music

# Or as environment variable
export BASE_PATH=/music
```

**Important:** 
- The base path MUST start with `/` (e.g., `/music`, not `music`)
- The base path should NOT end with `/` (e.g., `/music`, not `/music/`)

### 2. Start the Web Server

The server will automatically use the base path:

```bash
./ya_music_web
```

You should see:
```
Starting web server on http://localhost:8080/music
```

## Reverse Proxy Configuration Examples

### Apache Configuration

Add this to your Apache configuration:

```apache
<VirtualHost *:80>
    ServerName example.com
    
    # Proxy the /music path to the Go application
    # Note: Both /music and /music/ work without redirects
    ProxyPass /music/ http://localhost:8080/music/
    ProxyPassReverse /music/ http://localhost:8080/music/
    ProxyPass /music http://localhost:8080/music
    ProxyPassReverse /music http://localhost:8080/music
    
    # Optional: Add headers
    ProxyPreserveHost On
    RequestHeader set X-Forwarded-Proto "http"
</VirtualHost>
```

For HTTPS:

```apache
<VirtualHost *:443>
    ServerName example.com
    
    SSLEngine on
    SSLCertificateFile /path/to/cert.pem
    SSLCertificateKeyFile /path/to/key.pem
    
    # Proxy the /music path to the Go application
    # Note: Both /music and /music/ work without redirects
    ProxyPass /music/ http://localhost:8080/music/
    ProxyPassReverse /music/ http://localhost:8080/music/
    ProxyPass /music http://localhost:8080/music
    ProxyPassReverse /music http://localhost:8080/music
    
    ProxyPreserveHost On
    RequestHeader set X-Forwarded-Proto "https"
</VirtualHost>
```

### Nginx Configuration

Add this to your Nginx configuration:

```nginx
server {
    listen 80;
    server_name example.com;
    
    # Handle both /music and /music/ paths
    location ~ ^/music/?$ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # Handle sub-paths like /music/api/*, /music/css/*, etc.
    location /music/ {
        proxy_pass http://localhost:8080/music/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

For HTTPS:

```nginx
server {
    listen 443 ssl;
    server_name example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # Handle both /music and /music/ paths
    location ~ ^/music/?$ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # Handle sub-paths like /music/api/*, /music/css/*, etc.
    location /music/ {
        proxy_pass http://localhost:8080/music/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## How It Works

The application automatically handles base paths through several mechanisms:

### 1. HTML Base Tag
The server injects a `<base>` tag into the HTML:

```html
<head>
    <base href="/music/">
    <script>window.BASE_PATH = '/music';</script>
    ...
</head>
```

This ensures all relative URLs in the HTML are resolved correctly.

### 2. Relative URLs
All static resources and API calls use relative URLs:

- CSS: `css/styles.css` → `/music/css/styles.css`
- JavaScript: `js/app.js` → `/music/js/app.js`
- API: `api/search` → `/music/api/search`

### 3. Server Routing
The Go server automatically prefixes all routes with the base path and handles both with and without trailing slashes:

- Index: Both `/music` and `/music/` → serves HTML with injected base tag (no redirect)
- Static files: `/music/` → serves from `./static/`
- API endpoints: `/music/api/search` → API handler

**Note:** The application handles both `/music` and `/music/` without any redirects, making it compatible with various reverse proxy configurations.

### 4. Service Worker
The service worker automatically detects its base path from its own URL and adjusts cached resources accordingly.

## Testing Your Setup

### 1. Test Without Reverse Proxy

Start the server with base path:

```bash
BASE_PATH=/music ./ya_music_web
```

Open browser to `http://localhost:8080/music` - it should work!

### 2. Test with Reverse Proxy

After configuring your reverse proxy, access:
- `http://example.com/music` - should show the application
- Check browser console for any 404 errors on static files
- Try searching for music to verify API calls work

## Troubleshooting

### Path Doubling Issue (/music/music)

**Problem:** Accessing `/music` results in `/music/music` or causes path doubling.

**Solution:**
This was an issue in earlier versions where the app would redirect `/music` to `/music/`, which could cause problems with certain reverse proxy configurations. This has been fixed in the current version:

1. The application now handles both `/music` and `/music/` without any HTTP redirects
2. Update your reverse proxy configuration to handle both paths as shown in the examples above
3. Ensure your proxy configuration doesn't add extra path prefixes

If you're still experiencing this issue:
1. Verify you're using the latest version of the application
2. Check your reverse proxy configuration matches the examples above
3. Test accessing `http://localhost:8080/music` directly (without the proxy) to verify the app works correctly

### Static Files Return 404

**Problem:** CSS/JS files return 404 errors.

**Solution:** 
1. Verify `BASE_PATH` is set correctly (starts with `/`, no trailing `/`)
2. Ensure proxy configuration includes trailing slashes consistently
3. Check proxy logs to see what URLs are being requested

### API Calls Fail

**Problem:** API endpoints return errors.

**Solution:**
1. Verify the proxy is forwarding to the correct base path
2. Check that CORS headers are being set correctly
3. Look at browser Network tab to see actual request URLs

### Service Worker Issues

**Problem:** Service worker fails to cache resources.

**Solution:**
1. Clear browser cache and service worker
2. Hard refresh the page (Ctrl+Shift+R)
3. Check browser console for service worker errors

## Multiple Instances

You can run multiple instances with different base paths:

```bash
# Instance 1: /music
BASE_PATH=/music PORT=8080 ./ya_music_web &

# Instance 2: /radio  
BASE_PATH=/radio PORT=8081 ./ya_music_web &
```

Then configure your reverse proxy to route accordingly.

## Root Path Deployment

To deploy at the root path (no subpath), simply don't set `BASE_PATH` or set it to empty:

```bash
# No base path
./ya_music_web

# Or explicitly empty
BASE_PATH= ./ya_music_web
```

The application will work at `http://example.com/` instead of a subpath.
