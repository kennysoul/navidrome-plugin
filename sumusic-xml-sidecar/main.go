package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/plugins/pdk/go/metadata"
)

const (
	artistSidecarName = ".sumusic.artist.xml"
	albumSidecarName  = ".sumusic.album.xml"
)

type xmlSidecarPlugin struct{}

type sidecarArtist struct {
	XMLName   xml.Name `xml:"artist"`
	Name      string   `xml:"name"`
	Biography string   `xml:"biography"`
}

type sidecarAlbum struct {
	XMLName     xml.Name `xml:"album"`
	Name        string   `xml:"name"`
	Artist      string   `xml:"artist"`
	Description string   `xml:"description"`
	URL         string   `xml:"url"`
	MBID        string   `xml:"mbid"`
}

func init() {
	metadata.Register(&xmlSidecarPlugin{})
}

var (
	_ metadata.ArtistBiographyProvider = (*xmlSidecarPlugin)(nil)
	_ metadata.AlbumInfoProvider       = (*xmlSidecarPlugin)(nil)
)

func (p *xmlSidecarPlugin) GetArtistBiography(req metadata.ArtistRequest) (*metadata.ArtistBiographyResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("artist name is required")
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
			return &metadata.ArtistBiographyResponse{Biography: bio}, nil
		}
	}
	return nil, fmt.Errorf("artist sidecar not found")
}

func (p *xmlSidecarPlugin) GetAlbumInfo(req metadata.AlbumRequest) (*metadata.AlbumInfoResponse, error) {
	albumName := strings.TrimSpace(req.Name)
	artistName := strings.TrimSpace(req.Artist)
	if albumName == "" || artistName == "" {
		return nil, fmt.Errorf("album name and artist are required")
	}
	pattern := filepath.Join("/libraries", "*", sanitizeName(artistName), sanitizeName(albumName), albumSidecarName)
	matches, _ := filepath.Glob(pattern)
	for _, f := range matches {
		var side sidecarAlbum
		if err := readXMLFile(f, &side); err != nil {
			continue
		}
		resp := &metadata.AlbumInfoResponse{
			Name:        firstNonEmpty(strings.TrimSpace(side.Name), albumName),
			MBID:        strings.TrimSpace(side.MBID),
			Description: strings.TrimSpace(side.Description),
			URL:         strings.TrimSpace(side.URL),
		}
		if resp.Description != "" || resp.URL != "" || resp.MBID != "" {
			return resp, nil
		}
	}
	return nil, fmt.Errorf("album sidecar not found")
}

func readXMLFile(path string, out any) error {
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := xml.Unmarshal(buf, out); err != nil {
		return err
	}
	return nil
}

func sanitizeName(v string) string {
	v = strings.TrimSpace(v)
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return r.Replace(v)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func main() {}
