# EarAPI

A small HTTP API built with Gin exposing Steam, TMDB, Netflix Top 10, and YouTube playlist utilities.

## Run the server

Build and run:

```bash
go build -o earapi .
./earapi
```

The server listens on the port configured in `config/earapi.json` (field `api.port`). Example: if `api.port` is `8080`, your base URL is `http://localhost:8080`.

Production domain used below: `https://api.earentir.dev`.

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
curl -sS -X POST https://api.earentir.dev/youtube/v1/playlist/add -H 'Content-Type: application/json' -d '{"playlistName":"My Playlist","video":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","force":false,"user":"alice@example.com"}'
```

- Create a playlist

```bash
curl -sS -X POST https://api.earentir.dev/youtube/v1/playlist/create -H 'Content-Type: application/json' -d '{"name":"New Playlist Name","privacy":"private"}'
```

- List items in a playlist

```bash
curl -sS "https://api.earentir.dev/youtube/v1/playlist/items?name=My%20Playlist&fuzzy=false&metadata=false"
```

- List items with metadata (from additions log)

```bash
curl -sS "https://api.earentir.dev/youtube/v1/playlist/items?name=Old%20Shit%20%28up%20to%202000%29&metadata=true"
```

- Metadata for a specific video in a playlist

```bash
curl -sS "https://api.earentir.dev/youtube/v1/playlist/video/meta?name=My%20Playlist&videoId=dQw4w9WgXcQ"
```

## Netflix Top 10 Endpoints

Base: `/netflix/v1`

- Global top (movies default)

```bash
curl -sS "https://api.earentir.dev/netflix/v1/top" | jq '.[0:3]'
```

Example response (first 3):
```json
[
  {
    "rank": 1,
    "title": "The Wrong Paris",
    "poster": "https://dnm.nflximg.net/api/v6/E8vDc_W8CLv7-yMQu8KMEC7Rrr8/AAAABbhtP9t0JSAxeMh5rRoJnKhsomNNIiZ7rlAIF-1S56gmOh2p9vYND3JDc3xzWvc47QAMJpPL1ezCtXngPLCcPOaYnIdar102SzfT.jpg?r=5f4",
    "playUrl": "https://www.netflix.com/watch/81700163?trackId=276465340&trkId=276465340&src=tudum",
    "detailUrl": "https://www.netflix.com/tudum/the-wrong-paris"
  },
  {
    "rank": 2,
    "title": "KPop Demon Hunters",
    "poster": "https://dnm.nflximg.net/api/v6/E8vDc_W8CLv7-yMQu8KMEC7Rrr8/AAAABeKUv0SD_U6cZTjamXsfkdCqOJ5I0Tm0gQBHIxXe1zerjb5vqEHrSnwGaOyCGobgkUr_rk4zwadhQc8EXar20dGutosyCJmpMcOE.jpg?r=983",
    "playUrl": "https://www.netflix.com/watch/81498621?trackId=276465340&trkId=276465340&src=tudum",
    "detailUrl": "https://www.netflix.com/tudum/kpop-demon-hunters"
  },
  {
    "rank": 3,
    "title": "Ice Road: Vengeance",
    "poster": "https://dnm.nflximg.net/api/v6/E8vDc_W8CLv7-yMQu8KMEC7Rrr8/AAAABXdhWsMoLTViEtRxLfkXWxuJzKGsNh-NdvpQ4Qu3C8hpD-Y3FVfAx7i-86JdgZQhPrY0phFvgOy9CZeo2ml5--WVbNOim1N1wt6b.jpg?r=0a2",
    "playUrl": "",
    "detailUrl": ""
  }
]
```

- Global top TV

```bash
curl -sS "https://api.earentir.dev/netflix/v1/top?type=tv" | jq '.[0:3]'
```

Example response (first 3):
```json
[
  {
    "rank": 1,
    "title": "Wednesday: Season 2",
    "poster": "https://dnm.nflximg.net/api/v6/E8vDc_W8CLv7-yMQu8KMEC7Rrr8/AAAABfRVm_e-dJQu7tw2OB4l4Vmn5S0y687a8NfkhAQsBCsEV9jmBX_Gx2iH8jsAITtLrv05e-midL9rxTnywCyW8cdT-rpMh2dMsdrF.jpg?r=af7",
    "playUrl": "https://www.netflix.com/watch/81231974?trackId=276465340&trkId=276465340&src=tudum",
    "detailUrl": "https://www.netflix.com/tudum/wednesday"
  },
  {
    "rank": 2,
    "title": "Black Rabbit: Limited Series",
    "poster": "https://dnm.nflximg.net/api/v6/E8vDc_W8CLv7-yMQu8KMEC7Rrr8/AAAABfmU4ibUNL54p9fL-7aAhUEDPkHn2Io9pg5hLSPMOujR_NnBWE6DsI1_NmEmYmxbZMYCaSC7hk0znaCRY3ia2GUGMVB8pHOycK5_.jpg?r=271",
    "playUrl": "https://www.netflix.com/watch/81630027?trackId=276465340&trkId=276465340&src=tudum",
    "detailUrl": "https://www.netflix.com/tudum/black-rabbit"
  },
  {
    "rank": 3,
    "title": "Beauty in Black: Season 2",
    "poster": "https://dnm.nflximg.net/api/v6/E8vDc_W8CLv7-yMQu8KMEC7Rrr8/AAAABeh2gWsMZ9vtBf_EtUV-iEd6EDG2ttXguHNx43xY60jRetCA37M1bEhK143C-DiguA8b--yRY8nsd6S29_jF4nVy8_88JjOYAXTE.jpg?r=470",
    "playUrl": "https://www.netflix.com/watch/81764523?trackId=276465340&trkId=276465340&src=tudum",
    "detailUrl": "https://www.netflix.com/tudum/beauty-in-black"
  }
]
```

- Country-specific (movies) and optional week

```bash
curl -sS "https://api.earentir.dev/netflix/v1/top?country=us" | jq '.[0:3]'
curl -sS "https://api.earentir.dev/netflix/v1/top?country=us&week=2025-09-22" | jq '.[0:3]'
```

Example response (first 3):
```json
[
  {
    "rank": 1,
    "title": "The Wrong Paris",
    "poster": "https://dnm.nflximg.net/api/v6/E8vDc_W8CLv7-yMQu8KMEC7Rrr8/AAAABbhtP9t0JSAxeMh5rRoJnKhsomNNIiZ7rlAIF-1S56gmOh2p9vYND3JDc3xzWvc47QAMJpPL1ezCtXngPLCcPOaYnIdar102SzfT.jpg?r=5f4",
    "playUrl": "https://www.netflix.com/watch/81700163?trackId=276465340&trkId=276465340&src=tudum",
    "detailUrl": "https://www.netflix.com/tudum/the-wrong-paris"
  },
  {
    "rank": 2,
    "title": "KPop Demon Hunters",
    "poster": "https://dnm.nflximg.net/api/v6/E8vDc_W8CLv7-yMQu8KMEC7Rrr8/AAAABeKUv0SD_U6cZTjamXsfkdCqOJ5I0Tm0gQBHIxXe1zerjb5vqEHrSnwGaOyCGobgkUr_rk4zwadhQc8EXar20dGutosyCJmpMcOE.jpg?r=983",
    "playUrl": "https://www.netflix.com/watch/81498621?trackId=276465340&trkId=276465340&src=tudum",
    "detailUrl": "https://www.netflix.com/tudum/kpop-demon-hunters"
  },
  {
    "rank": 3,
    "title": "Ice Road: Vengeance",
    "poster": "https://dnm.nflximg.net/api/v6/E8vDc_W8CLv7-yMQu8KMEC7Rrr8/AAAABXdhWsMoLTViEtRxLfkXWxuJzKGsNh-NdvpQ4Qu3C8hpD-Y3FVfAx7i-86JdgZQhPrY0phFvgOy9CZeo2ml5--WVbNOim1N1wt6b.jpg?r=0a2",
    "playUrl": "",
    "detailUrl": ""
  }
]
```

- Most popular TV

```bash
curl -sS "https://api.earentir.dev/netflix/v1/top?type=popular" | jq '.[0:3]'
```

Example response (first 3):
```json
[
  {
    "rank": 1,
    "title": "Wednesday: Season 1",
    "poster": "https://dnm.nflximg.net/api/v6/E8vDc_W8CLv7-yMQu8KMEC7Rrr8/AAAABWu3yLC_h4YnQ2jlZhvhKuYdGi8rl6YJHeforrlpsq_3gs4wWR4ZLmvBoE2ASNm05DYZ3fnXeiyb5nZwotSgxDhNwumquixbOxdF.jpg?r=570",
    "playUrl": "https://www.netflix.com/watch/81231974?trackId=276465340&trkId=276465340&src=tudum",
    "detailUrl": "https://www.netflix.com/tudum/wednesday"
  },
  {
    "rank": 2,
    "title": "Adolescence: Limited Series",
    "poster": "https://dnm.nflximg.net/api/v6/E8vDc_W8CLv7-yMQu8KMEC7Rrr8/AAAABWZHB3sXiJecf3V_R5D5vsFSH4C2iK0CzlVU2HyNE-NsNXn1h_l3MLEIDGuhn08nQENHwkaG_eNtrpjC0Fu4bC5-yHLHpEHDGbyO.jpg?r=0c4",
    "playUrl": "https://www.netflix.com/watch/81756069?trackId=276465340&trkId=276465340&src=tudum",
    "detailUrl": "https://www.netflix.com/tudum/adolescence"
  },
  {
    "rank": 3,
    "title": "Stranger Things 4",
    "poster": "https://dnm.nflximg.net/api/v6/E8vDc_W8CLv7-yMQu8KMEC7Rrr8/AAAABaNeUcpwO2JirpMHb6WJugLqDAgzWFGjr-Zyvd3ppzo3trrc2wE5U_zBVhJI89FlxVg5ZPGPeUY8mXAipDMZ1Va1fzGLZ7BEzwda.jpg?r=b45",
    "playUrl": "https://www.netflix.com/watch/80057281?trackId=276465340&trkId=276465340&src=tudum",
    "detailUrl": "https://www.netflix.com/tudum/stranger-things"
  }
]
```

## TMDB Endpoints

Base: `/tmdb/v1`

- Search movies by query

```bash
curl -sS "https://api.earentir.dev/tmdb/v1/search?q=Blade%20Runner" | jq '.'
```

Example response:
```json
{
  "info": {
    "ID": 78,
    "Adult": false,
    "Title": "Blade Runner",
    "Name": "",
    "Overview": "In the smog-choked dystopian Los Angeles of 2019, blade runner Rick Deckard is called out of retirement to terminate a quartet of replicants who have escaped to Earth seeking their creator for a way to extend their short life spans.",
    "PosterPath": "/63N9uy8nd9j7Eog2axPQ8lbr3Wj.jpg",
    "ProfilePath": "",
    "FirstAirDate": "",
    "ReleaseDate": "1982-06-25",
    "OriginCountry": null,
    "OriginalLanguage": "en",
    "MediaType": "movie"
  },
  "movie": "Blade Runner"
}
```

```bash
curl -sS "https://api.earentir.dev/tmdb/v1/search?query=The%20Matrix" | jq '.'
```

Example response:
```json
{
  "info": {
    "ID": 603,
    "Adult": false,
    "Title": "The Matrix",
    "Name": "",
    "Overview": "Set in the 22nd century, The Matrix tells the story of a computer hacker who joins a group of underground insurgents fighting the vast and powerful computers who now rule the earth.",
    "PosterPath": "/p96dm7sCMn4VYAStA6siNz30G1r.jpg",
    "ProfilePath": "",
    "FirstAirDate": "",
    "ReleaseDate": "1999-03-31",
    "OriginCountry": null,
    "OriginalLanguage": "en",
    "MediaType": "movie"
  },
  "movie": "The Matrix"
}
```

## Steam Endpoints

Base: `/steam/v1`

- Get top apps for a user (by playtime)

```bash
curl -sS "https://api.earentir.dev/steam/v1/top?userid=76561198011985757&count=5&sortby=playtime" | jq '.'
```

Example response (first 5):
```json
{
  "data": [
    {
      "appid": 218230,
      "name": "PlanetSide 2",
      "playtime_forever": 111056,
      "rtime_last_played": 1699288243
    },
    {
      "appid": 227940,
      "name": "Heroes & Generals",
      "playtime_forever": 36213,
      "rtime_last_played": 1564853631
    },
    {
      "appid": 476620,
      "name": "Call of Duty: WWII - Multiplayer",
      "playtime_forever": 31851,
      "rtime_last_played": 1539278329
    },
    {
      "appid": 578080,
      "name": "PUBG: BATTLEGROUNDS",
      "playtime_forever": 23574,
      "rtime_last_played": 1723742324
    },
    {
      "appid": 252490,
      "name": "Rust",
      "playtime_forever": 20552,
      "rtime_last_played": 1673644011
    }
  ],
  "msg": "",
  "success": true
}
```

- Get a user’s numeric Steam ID from vanity name

```bash
curl -sS "https://api.earentir.dev/steam/v1/getuserid?username=earentir" | jq '.'
```

Example response:
```json
{
  "data": {
    "steamID": "76561198011985757"
  },
  "msg": "",
  "success": true
}
```

- Apps used by a user

```bash
curl -sS "https://api.earentir.dev/steam/v1/appsused?userid=76561198011985757" | jq '.data[0:3]'
```

Example response (first 3):
```json
[
  {
    "appid": 10,
    "name": "Counter-Strike",
    "playtime_forever": 15,
    "rtime_last_played": 86400
  },
  {
    "appid": 20,
    "name": "Team Fortress Classic",
    "playtime_forever": 0,
    "rtime_last_played": 0
  },
  {
    "appid": 30,
    "name": "Day of Defeat",
    "playtime_forever": 0,
    "rtime_last_played": 0
  }
]
```

- App details by appid

```bash
curl -sS "https://api.earentir.dev/steam/v1/appdata?appid=1086940" | jq '.'
```

Example response (truncated):
```json
{
  "data": {
    "appid": 1086940,
    "name": "Baldur's Gate 3",
    "storeurl": "https://store.steampowered.com/app/1086940/",
    "price": { "currency": "EUR", "final": 5999 },
    "type": "game"
  },
  "msg": "",
  "success": true
}
```

- Search app by name

```bash
curl -sS "https://api.earentir.dev/steam/v1/search?app=Baldur%27s%20Gate%203" | jq '.'
```

Example response:
```json
{
  "data": {
    "app": "1086940"
  },
  "msg": "",
  "success": true
}
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
