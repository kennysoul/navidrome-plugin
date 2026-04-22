# SuMusic XML Sidecar Metadata Plugin for Navidrome

A Navidrome metadata plugin that reads artist and album information from XML sidecar files placed alongside your music library files.

## Features

- Reads artist biography from `.sumusic.artist.xml` sidecar files
- Reads album info (description, URL, MBID) from `.sumusic.album.xml` sidecar files
- Works with Navidrome's plugin system (WASM-based)

## Installation

Download the latest `sumusic-xml-sidecar.ndp` from the [Releases](https://github.com/kennysoul/navidrome-plugin/releases) page, then place it in your Navidrome plugins directory.

## Sidecar File Format

### Artist: `.sumusic.artist.xml`

Place in the artist folder, e.g. `/music/Artist Name/.sumusic.artist.xml`

```xml
<artist>
  <name>Artist Name</name>
  <biography>Artist biography text...</biography>
</artist>
```

### Album: `.sumusic.album.xml`

Place in the album folder, e.g. `/music/Artist Name/Album Name/.sumusic.album.xml`

```xml
<album>
  <name>Album Name</name>
  <artist>Artist Name</artist>
  <description>Album description...</description>
  <url>https://example.com/album</url>
  <mbid>optional-musicbrainz-id</mbid>
</album>
```

## Building Locally

Requirements: Go 1.22+, TinyGo 0.35.0+

```bash
cd sumusic-xml-sidecar
go mod download
tinygo build -o plugin.wasm -target wasi -scheduler none -no-debug ./...
zip sumusic-xml-sidecar.ndp plugin.wasm manifest.json
```

## CI

GitHub Actions automatically builds the plugin on every push using Go + TinyGo (installed separately to avoid the missing `go` binary issue with the official TinyGo Docker image).
