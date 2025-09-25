# EarAPI

A small HTTP API built with Gin exposing Steam, TMDB, Netflix Top 10, and YouTube playlist utilities.

## Run the server

Build and run:

```bash
go build -o earapi .
./earapi
```

The server listens on the port configured in `config/earapi.json` (field `api.port`). Example: if `api.port` is `8080`, your base URL is `http://localhost:8080`.

Production example domain used below: `https://api.domain.tld`.

## YouTube OAuth (headless)

Two supported flows (use one):

- Device Flow (recommended for headless)

```bash
./earapi --youtube-auth-device
# It prints:
# Visit: https://... (verification URL)
# Enter code: ABCD-EFGH
# Approve in a browser; the app saves the refresh token to config/earapi.json
```

- Legacy OOB helpers (deprecated by Google, may fail):

```bash
./earapi --youtube-auth-url
# prints a URL

./earapi --youtube-auth-code "PASTE_CODE"
# saves the refresh token if successful
```

Ensure YouTube Data API v3 is enabled and the OAuth client is appropriate for the flow you choose.

## YouTube Endpoints

Base: `/youtube/v1`

- Add a video to a playlist

```bash
curl -sS -X POST https://api.domain.tld/youtube/v1/playlist/add -H 'Content-Type: application/json' -d '{"playlistName":"My Playlist","video":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","force":false,"user":"alice@example.com"}'
```

- Create a playlist

```bash
curl -sS -X POST https://api.domain.tld/youtube/v1/playlist/create -H 'Content-Type: application/json' -d '{"name":"New Playlist Name","privacy":"private"}'
```

- List items in a playlist

```bash
curl -sS "https://api.domain.tld/youtube/v1/playlist/items?name=My%20Playlist&fuzzy=false&metadata=false"
```

- List items with metadata (from additions log)

```bash
curl -sS "https://api.domain.tld/youtube/v1/playlist/items?name=Old%20Shit%20%28up%20to%202000%29&metadata=true"
```

- Metadata for a specific video in a playlist

```bash
curl -sS "https://api.domain.tld/youtube/v1/playlist/video/meta?name=My%20Playlist&videoId=dQw4w9WgXcQ"
```

## Netflix Top 10 Endpoints

Base: `/netflix/v1`

- Global top (movies default)

```bash
curl -sS "https://api.domain.tld/netflix/v1/top"
```

- Global top TV

```bash
curl -sS "https://api.domain.tld/netflix/v1/top?type=tv"
```

- Country-specific (movies) and optional week

```bash
curl -sS "https://api.domain.tld/netflix/v1/top?country=us"
curl -sS "https://api.domain.tld/netflix/v1/top?country=us&week=2025-09-22"
```

- Most popular TV

```bash
curl -sS "https://api.domain.tld/netflix/v1/top?type=popular"
```

## TMDB Endpoints

Base: `/tmdb/v1`

- Search movies by query

```bash
curl -sS "https://api.domain.tld/tmdb/v1/search?q=The%20Matrix"
```

## Steam Endpoints

Base: `/steam/v1`

- Get top apps for a user (by playtime)

```bash
curl -sS "https://api.domain.tld/steam/v1/top?userid=76561198011985757&count=10&sortby=playtime"
```

- Get a user’s numeric Steam ID from vanity name

```bash
curl -sS "https://api.domain.tld/steam/v1/getuserid?username=earentir"
```

- Apps used by a user

```bash
curl -sS "https://api.domain.tld/steam/v1/appsused?userid=76561198011985757"
```

- App details by appid

```bash
curl -sS "https://api.domain.tld/steam/v1/appdata?appid=1086940"
```

- Search app by name

```bash
curl -sS "https://api.domain.tld/steam/v1/search?app=Baldur%27s%20Gate%203"
```

## Server configuration

`config/earapi.json` controls runtime settings, e.g.:

```json
{
  "api": { "port": "8080" },
  "apikeys": {
    "steamapikey": "YOUR_STEAM_KEY",
    "tmdbapitoken": "YOUR_TMDB_TOKEN"
  },
  "youtube": {
    "client_id": "GOOGLE_OAUTH_CLIENT_ID",
    "client_secret": "GOOGLE_OAUTH_CLIENT_SECRET",
    "refresh_token": "SAVED_REFRESH_TOKEN",
    "default_channel_id": "UCxxxxxxxxxxxxxxxx",
    "cache_minutes": 10
  }
}
```

- For YouTube, set `client_id`/`client_secret` for your OAuth client.
- Use `--youtube-auth-device` to obtain and persist `refresh_token`.
- Ensure required third-party APIs (YouTube Data API v3, Steam Web API, etc.) are enabled and keys configured.