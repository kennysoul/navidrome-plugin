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

// --- XML sidecar types (two supported formats) ---

const (
	artistSidecarName = ".sumusic.artist.xml"
	albumSidecarName  = ".sumusic.album.xml"
)

// Format A: <artist><biography>...</biography></artist>
type sidecarArtist struct {
	XMLName   xml.Name `xml:"artist"`
	Biography string   `xml:"biography"`
}

// Format B (sumusic native): <artistProfile><name>...</name><bio>...</bio><ids>...</ids></artistProfile>
type sidecarArtistID struct {
	Platform string `xml:"platform,attr"`
	Value    string `xml:",chardata"`
}

type sidecarArtistProfile struct {
	XMLName xml.Name          `xml:"artistProfile"`
	Name    string            `xml:"name"`
	Bio     string            `xml:"bio"`
	IDs     []sidecarArtistID `xml:"ids>id"`
}

func (p *sidecarArtistProfile) musicBrainzID() string {
	for _, id := range p.IDs {
		if id.Platform == "musicbrainz" {
			return strings.TrimSpace(id.Value)
		}
	}
	return ""
}

type sidecarAlbum struct {
	XMLName     xml.Name `xml:"album"`
	Description string   `xml:"description"`
	URL         string   `xml:"url"`
	MBID        string   `xml:"mbid"`
}

// artistSidecarInfo holds the parsed result regardless of XML format.
type artistSidecarInfo struct {
	Bio  string
	Name string // populated by Format B only
	MBID string // populated by Format B only
}

// parseArtistSidecar reads a sidecar file and returns its info,
// supporting both Format A (<artist>) and Format B (<artistProfile>).
func parseArtistSidecar(path string) (artistSidecarInfo, bool) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return artistSidecarInfo{}, false
	}
	// Try Format B first: richer structure with <name> and MBID support.
	var b sidecarArtistProfile
	if xml.Unmarshal(buf, &b) == nil {
		if bio := strings.TrimSpace(b.Bio); bio != "" {
			return artistSidecarInfo{
				Bio:  bio,
				Name: strings.TrimSpace(b.Name),
				MBID: b.musicBrainzID(),
			}, true
		}
	}
	// Try Format A: <artist><biography>
	var a sidecarArtist
	if xml.Unmarshal(buf, &a) == nil {
		if bio := strings.TrimSpace(a.Biography); bio != "" {
			return artistSidecarInfo{Bio: bio}, true
		}
	}
	return artistSidecarInfo{}, false
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
	mbid := strings.TrimSpace(req.MBID)
	if name == "" {
		extism.SetError(fmt.Errorf("artist name is required"))
		return -1
	}

	// Pass 1: exact directory-name match (fast path).
	pattern := filepath.Join("/libraries", "*", sanitizeName(name), artistSidecarName)
	if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
		for _, f := range matches {
			if info, ok := parseArtistSidecar(f); ok {
				return outputArtistBio(info.Bio)
			}
		}
	}

	// Pass 2: scan all artist directories.
	// This handles cases where the directory name uses a different CJK
	// character variant from the Navidrome-stored artist name (e.g. 惠 vs 恵).
	// Matches by MBID (most reliable) or by the <name> field in the sidecar.
	allPattern := filepath.Join("/libraries", "*", "*", artistSidecarName)
	allMatches, _ := filepath.Glob(allPattern)
	for _, f := range allMatches {
		info, ok := parseArtistSidecar(f)
		if !ok {
			continue
		}
		byMBID := mbid != "" && info.MBID != "" && info.MBID == mbid
		byName := info.Name != "" && strings.EqualFold(info.Name, name)
		if byMBID || byName {
			return outputArtistBio(info.Bio)
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

	// Pass 1: exact artist/album path match.
	pattern := filepath.Join("/libraries", "*", sanitizeName(artistName), sanitizeName(albumName), albumSidecarName)
	if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
		for _, f := range matches {
			if resp, ok := parseAlbumSidecar(f, albumName); ok {
				if err := extism.OutputJSON(resp); err != nil {
					extism.SetError(err)
					return -1
				}
				return 0
			}
		}
	}

	// Pass 2: wildcard artist directory to handle character-variant mismatches.
	allPattern := filepath.Join("/libraries", "*", "*", sanitizeName(albumName), albumSidecarName)
	allMatches, _ := filepath.Glob(allPattern)
	for _, f := range allMatches {
		if resp, ok := parseAlbumSidecar(f, albumName); ok {
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

func outputArtistBio(bio string) int32 {
	resp := artistBiographyResponse{Biography: bio}
	if err := extism.OutputJSON(resp); err != nil {
		extism.SetError(err)
		return -1
	}
	return 0
}

func parseAlbumSidecar(path string, albumName string) (albumInfoResponse, bool) {
	var side sidecarAlbum
	if err := readXMLFile(path, &side); err != nil {
		return albumInfoResponse{}, false
	}
	resp := albumInfoResponse{
		Name:        albumName,
		MBID:        strings.TrimSpace(side.MBID),
		Description: strings.TrimSpace(side.Description),
		URL:         strings.TrimSpace(side.URL),
	}
	if resp.Description != "" || resp.URL != "" || resp.MBID != "" {
		return resp, true
	}
	return albumInfoResponse{}, false
}

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

