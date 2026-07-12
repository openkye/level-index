package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Index struct {
	SchemaVersion int        `json:"schemaVersion"`
	GeneratedAt   string     `json:"generatedAt"`
	Description   string     `json:"description,omitempty"`
	Sources       []Source   `json:"sources,omitempty"`
	LevelSets     []LevelSet `json:"levelSets"`
}

type Source struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	URL   string `json:"url"`
	Notes string `json:"notes,omitempty"`
}

type LevelSet struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Author      string      `json:"author,omitempty"`
	SHA256      string      `json:"sha256"`
	Bytes       int64       `json:"bytes,omitempty"`
	LevelCount  int         `json:"levelCount,omitempty"`
	FileName    string      `json:"fileName,omitempty"`
	Description string      `json:"description,omitempty"`
	License     string      `json:"license,omitempty"`
	SourceIDs   []string    `json:"sourceIds,omitempty"`
	URLs        []URLRecord `json:"urls"`
	Notes       string      `json:"notes,omitempty"`
}

type URLRecord struct {
	URL         string `json:"url"`
	SourceID    string `json:"sourceId,omitempty"`
	ArchiveURL  string `json:"archiveUrl,omitempty"`
	RetrievedAt string `json:"retrievedAt,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

var (
	idRE     = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	sourceRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	hashRE   = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "validate":
		fs := flag.NewFlagSet("validate", flag.ExitOnError)
		indexPath := fs.String("index", "index.json", "path to index JSON")
		_ = fs.Parse(os.Args[2:])
		idx, err := readIndex(*indexPath)
		if err != nil {
			exitErr(err)
		}
		if err := validate(idx); err != nil {
			exitErr(err)
		}
		fmt.Printf("validated %d level sets\n", len(idx.LevelSets))
	case "hash":
		fs := flag.NewFlagSet("hash", flag.ExitOnError)
		_ = fs.Parse(os.Args[2:])
		if fs.NArg() == 0 {
			exitErr(errors.New("hash requires at least one file path"))
		}
		for _, path := range fs.Args() {
			hash, size, err := hashFile(path)
			if err != nil {
				exitErr(err)
			}
			fmt.Printf("%s  %d  %s\n", hash, size, path)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: kyeindex validate [-index index.json] | kyeindex hash FILE...")
}

func readIndex(path string) (Index, error) {
	var idx Index
	f, err := os.Open(path)
	if err != nil {
		return idx, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&idx); err != nil {
		return idx, err
	}
	return idx, nil
}

func validate(idx Index) error {
	if idx.SchemaVersion != 1 {
		return fmt.Errorf("schemaVersion must be 1, got %d", idx.SchemaVersion)
	}
	if _, err := time.Parse(time.RFC3339, idx.GeneratedAt); err != nil {
		return fmt.Errorf("generatedAt must be RFC3339: %w", err)
	}

	sources := map[string]bool{}
	for _, source := range idx.Sources {
		if !sourceRE.MatchString(source.ID) {
			return fmt.Errorf("invalid source id %q", source.ID)
		}
		if sources[source.ID] {
			return fmt.Errorf("duplicate source id %q", source.ID)
		}
		sources[source.ID] = true
		if source.Name == "" {
			return fmt.Errorf("source %q has empty name", source.ID)
		}
		if err := requireURL(source.URL); err != nil {
			return fmt.Errorf("source %q url: %w", source.ID, err)
		}
	}

	ids := map[string]bool{}
	hashes := map[string]string{}
	for _, set := range idx.LevelSets {
		if !idRE.MatchString(set.ID) {
			return fmt.Errorf("invalid level set id %q", set.ID)
		}
		if ids[set.ID] {
			return fmt.Errorf("duplicate level set id %q", set.ID)
		}
		ids[set.ID] = true
		if set.Title == "" {
			return fmt.Errorf("level set %q has empty title", set.ID)
		}
		if !hashRE.MatchString(set.SHA256) {
			return fmt.Errorf("level set %q has invalid sha256", set.ID)
		}
		if existing, ok := hashes[set.SHA256]; ok {
			return fmt.Errorf("sha256 %s used by both %q and %q", set.SHA256, existing, set.ID)
		}
		hashes[set.SHA256] = set.ID
		if set.Bytes < 0 {
			return fmt.Errorf("level set %q has negative bytes", set.ID)
		}
		if set.LevelCount < 0 {
			return fmt.Errorf("level set %q has negative levelCount", set.ID)
		}
		if set.FileName != "" && !strings.HasSuffix(strings.ToLower(set.FileName), ".kye") {
			return fmt.Errorf("level set %q fileName must end in .kye", set.ID)
		}
		for _, sourceID := range set.SourceIDs {
			if !sources[sourceID] {
				return fmt.Errorf("level set %q references unknown source %q", set.ID, sourceID)
			}
		}
		if len(set.URLs) == 0 {
			return fmt.Errorf("level set %q must have at least one url", set.ID)
		}
		seenURLs := map[string]bool{}
		for _, rec := range set.URLs {
			if err := requireURL(rec.URL); err != nil {
				return fmt.Errorf("level set %q url: %w", set.ID, err)
			}
			if seenURLs[rec.URL] {
				return fmt.Errorf("level set %q has duplicate url %q", set.ID, rec.URL)
			}
			seenURLs[rec.URL] = true
			if rec.SourceID != "" && !sources[rec.SourceID] {
				return fmt.Errorf("level set %q url references unknown source %q", set.ID, rec.SourceID)
			}
			if rec.ArchiveURL != "" {
				if err := requireURL(rec.ArchiveURL); err != nil {
					return fmt.Errorf("level set %q archiveUrl: %w", set.ID, err)
				}
			}
			if rec.RetrievedAt != "" {
				if _, err := time.Parse(time.RFC3339, rec.RetrievedAt); err != nil {
					return fmt.Errorf("level set %q retrievedAt must be RFC3339: %w", set.ID, err)
				}
			}
		}
	}

	if !sort.SliceIsSorted(idx.LevelSets, func(i, j int) bool { return idx.LevelSets[i].ID < idx.LevelSets[j].ID }) {
		return errors.New("levelSets must be sorted by id")
	}
	return nil
}

func requireURL(raw string) error {
	if raw == "" {
		return errors.New("empty URL")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL %q must use http or https", raw)
	}
	if parsed.Host == "" {
		return fmt.Errorf("URL %q has no host", raw)
	}
	return nil
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
