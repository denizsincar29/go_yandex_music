# Web Server Tests

This directory contains comprehensive tests for the Yandex Music web server.

## Running Tests

### Unit Tests Only

To run unit tests without requiring API credentials:

```bash
cd cmd/web
go test -v
```

### Integration Tests

Integration tests require valid Yandex Music API credentials. Set these environment variables:

```bash
export YA_MUSIC_TOKEN="your_token_here"
export YA_MUSIC_ID="your_user_id"
```

Then run:

```bash
cd cmd/web
go test -v
```

The integration tests will automatically run when credentials are available.

## Test Coverage

### Unit Tests

These tests don't require API credentials and verify:

- **Input Validation**: Missing or invalid parameters
- **Error Handling**: Proper HTTP status codes and error messages
- **CORS Middleware**: Correct CORS headers
- **Response Structures**: JSON encoding/decoding

### Integration Tests

These tests require API credentials and verify:

- **Search Functionality**: Real searches against Yandex Music API
- **Album Tracks**: Loading tracks from albums
- **API Integration**: End-to-end functionality with real data

## Test Files

- `web_server_test.go`: All tests for the web server handlers

## Known Issues

### Album Tracks Bug

The `TestAlbumTracksIntegration` test documents a known issue where the album tracks endpoint may return 0 tracks for some albums. This happens because:

1. The endpoint searches for tracks by album name
2. It filters results by album ID
3. The search may not return all tracks from the album

**Workaround**: The improved implementation now:
- Searches multiple pages of results
- Uses a map to deduplicate tracks
- Logs detailed information for debugging

## Adding New Tests

When adding new handlers or features:

1. Add unit tests for input validation
2. Add unit tests for error cases
3. Add integration tests if the feature requires API calls
4. Use `t.Skip()` for integration tests when credentials aren't available

Example:

```go
func TestNewFeature(t *testing.T) {
    if os.Getenv("YA_MUSIC_TOKEN") == "" {
        t.Skip("Skipping integration test: credentials not set")
    }
    // ... test code
}
```

## Continuous Integration

For CI/CD pipelines, run unit tests without credentials:

```bash
go test -v -short
```

Or explicitly skip integration tests by checking for the absence of environment variables.
