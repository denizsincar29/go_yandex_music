# Modern Proxy Header Support

## Overview

As of this version, the application supports the **X-Forwarded-Prefix** header, which is the modern, cloud-native way to handle base paths - similar to how FastAPI/Uvicorn and other modern web frameworks work.

## How It Works

### Traditional Approach (Still Supported)
```bash
BASE_PATH=/music ./ya_music_web
```
- Static base path set at startup
- Requires restart to change
- Works without reverse proxy

### Modern Approach (Recommended)
Your reverse proxy sends `X-Forwarded-Prefix` header:
```
X-Forwarded-Prefix: /music
```
- Dynamic base path per request
- No restart needed to change paths
- Cloud-native pattern
- **Enabled by default**

## Configuration Examples

### Nginx
```nginx
location /music/ {
    proxy_pass http://localhost:8080/;
    proxy_set_header X-Forwarded-Prefix /music;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

### Apache
```apache
<Location /music>
    ProxyPass http://localhost:8080/
    ProxyPassReverse http://localhost:8080/
    RequestHeader set X-Forwarded-Prefix "/music"
    ProxyPreserveHost On
</Location>
```

### Traefik
```yaml
http:
  middlewares:
    music-prefix:
      addPrefix:
        prefix: /music
      headers:
        customRequestHeaders:
          X-Forwarded-Prefix: /music
```

### Caddy
```
handle_path /music/* {
    header X-Forwarded-Prefix /music
    reverse_proxy localhost:8080
}
```

## Environment Variables

### USE_PROXY_HEADERS
- **Default**: `true`
- **Purpose**: Enable/disable X-Forwarded-Prefix header support
- **Example**: `USE_PROXY_HEADERS=false ./ya_music_web`

### BASE_PATH
- **Default**: `""` (root)
- **Purpose**: Static fallback when X-Forwarded-Prefix is not present
- **Example**: `BASE_PATH=/music ./ya_music_web`

## Behavior

The application checks headers in this order:

1. **If USE_PROXY_HEADERS=true** (default):
   - Check `X-Forwarded-Prefix` header
   - If present, use it as base path
   - If not present, fall back to `BASE_PATH`

2. **If USE_PROXY_HEADERS=false**:
   - Only use static `BASE_PATH` from environment

## Examples

### Example 1: Pure X-Forwarded-Prefix (No BASE_PATH)
```bash
# Start server at root
./ya_music_web

# Nginx adds header
# Request to example.com/music → X-Forwarded-Prefix: /music
# App dynamically serves with /music base path
```

### Example 2: X-Forwarded-Prefix with BASE_PATH Fallback
```bash
# Start server with fallback
BASE_PATH=/api ./ya_music_web

# With header: uses /music from X-Forwarded-Prefix
# Without header: uses /api from BASE_PATH
```

### Example 3: Disable Proxy Headers (Legacy Mode)
```bash
# Only use static BASE_PATH
BASE_PATH=/music USE_PROXY_HEADERS=false ./ya_music_web

# Ignores X-Forwarded-Prefix header
# Always uses /music
```

## Why This Approach?

This follows the **2025 best practice** for Go web applications:

✅ **Cloud-Native**: Works like FastAPI, Express, and other modern frameworks  
✅ **Dynamic**: Change base paths without restarting  
✅ **Flexible**: Falls back to static config when needed  
✅ **Simple**: Reverse proxy handles the complexity  
✅ **Backwards Compatible**: Static BASE_PATH still works  

## Comparison with Other Frameworks

### FastAPI/Uvicorn
```python
# Uvicorn automatically reads X-Forwarded-Prefix
uvicorn app:app --proxy-headers
```

### Express.js
```javascript
// Express with helmet
app.use(helmet({ contentSecurityPolicy: false }));
app.set('trust proxy', true);
```

### This App
```bash
# Enabled by default, just like modern frameworks
./ya_music_web
```

## Security Note

When using `X-Forwarded-Prefix`, ensure your reverse proxy is properly configured:

1. **Strip the header** from incoming requests (don't trust client)
2. **Add your own header** with the correct value
3. **Use a trusted proxy** (Nginx, Apache, Traefik, etc.)

Example Nginx config:
```nginx
proxy_set_header X-Forwarded-Prefix "";  # Clear any client header
proxy_set_header X-Forwarded-Prefix /music;  # Set your own
```

## Migration from BASE_PATH Only

If you're currently using only `BASE_PATH`:

### No Changes Needed!
Your current setup still works. The app now also supports X-Forwarded-Prefix as a bonus.

### To Use X-Forwarded-Prefix:
1. Update your reverse proxy config to send the header
2. Optionally remove `BASE_PATH` from your environment
3. Restart your proxy (not the app!)

That's it! No code changes, no app restart needed after initial setup.
