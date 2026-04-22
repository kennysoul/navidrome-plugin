//go:build wasip1

package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	extism "github.com/extism/go-pdk"
)

// --- Navidrome PDK types (inlined from navidrome/plugins/pdk/go/metadata) ---

const notImplementedCode int32 = -2

type artistRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	MBID string `json:"mbid,omitempty"`
}

type artistBiographyResponse struct {
	Biography string `json:"biography"`
}

type albumRequest struct {
	Name   string `json:"name"`
	Artist string `json:"artist"`
	MBID   string `json:"mbid,omitempty"`
}

type albumInfoResponse struct {
	Name        string `json:"name"`
	MBID        string `json:"mbid"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

// --- XML sidecar types ---

const (
	artistSidecarName = ".sumusic.artist.xml"
	albumSidecarName  = ".sumusic.album.xml"
)

type sidecarArtist struct {
	XMLName   xml.Name `xml:"artist"`
	Biography string   `xml:"biography"`
}

type sidecarAlbum struct {
	XMLName     xml.Name `xml:"album"`
	Description string   `xml:"description"`
	URL         string   `xml:"url"`
	MBID        string   `xml:"mbid"`
}

// --- WASM exports (navidrome MetadataAgent capability) ---

//go:wasmexport nd_get_artist_biography
func ndGetArtistBiography() int32 {
	var req artistRequest
	if err := extism.InputJSON(&req); err != nil {
		extism.SetError(err)
		return -1
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		extism.SetError(fmt.Errorf("artist name is required"))
		return -1
	}
	pattern := filepath.Join("/libraries", "*", sanitizeName(name), artistSidecarName)
	matches, _ := filepath.Glob(pattern)
	for _, f := range matches {
		var side sidecarArtist
		if err := readXMLFile(f, &side); err != nil {
			continue
		}
		bio := strings.TrimSpace(side.Biography)
		if bio != "" {
			resp := artistBiographyResponse{Biography: bio}
			if err := extism.OutputJSON(resp); err != nil {
				extism.SetError(err)
				return -1
			}
			return 0
		}
	}
	extism.SetError(fmt.Errorf("artist sidecar not found for: %s", name))
	return -1
}

//go:wasmexport nd_get_album_info
func ndGetAlbumInfo() int32 {
	var req albumRequest
	if err := extism.InputJSON(&req); err != nil {
		extism.SetError(err)
		return -1
	}
	albumName := strings.TrimSpace(req.Name)
	artistName := strings.TrimSpace(req.Artist)
	if albumName == "" || artistName == "" {
		extism.SetError(fmt.Errorf("album name and artist are required"))
		return -1
	}
	pattern := filepath.Join("/libraries", "*", sanitizeName(artistName), sanitizeName(albumName), albumSidecarName)
	matches, _ := filepath.Glob(pattern)
	for _, f := range matches {
		var side sidecarAlbum
		if err := readXMLFile(f, &side); err != nil {
			continue
		}
		resp := albumInfoResponse{
			Name:        albumName,
			MBID:        strings.TrimSpace(side.MBID),
			Description: strings.TrimSpace(side.Description),
			URL:         strings.TrimSpace(side.URL),
		}
		if resp.Description != "" || resp.URL != "" || resp.MBID != "" {
			if err := extism.OutputJSON(resp); err != nil {
				extism.SetError(err)
				return -1
			}
			return 0
		}
	}
	extism.SetError(fmt.Errorf("album sidecar not found for: %s / %s", artistName, albumName))
	return -1
}

// --- Helpers ---

func readXMLFile(path string, out any) error {
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return xml.Unmarshal(buf, out)
}

func sanitizeName(v string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return r.Replace(strings.TrimSpace(v))
}

func main() {}

