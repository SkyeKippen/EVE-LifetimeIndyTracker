// Package sde reads what each item refines into from EVE's static data
// export (SDE). ESI has no route for refine materials, so the app keeps the
// SDE's typeMaterials.jsonl in data/sde, downloading it from CCP the first
// time it's missing.
package sde

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

// TypeMaterialsPath is where the refine materials are kept, relative to the
// working directory like config/.env.
const TypeMaterialsPath = "data/sde/typeMaterials.jsonl"

// defaultSdeURL is CCP's latest SDE in JSON Lines; SDE_URL in config/.env
// overrides it.
const defaultSdeURL = "https://developers.eveonline.com/static-data/eve-online-static-data-latest-jsonl.zip"

// ErrNotLoaded means the refine materials haven't been read yet, e.g. while
// the SDE is still downloading.
var ErrNotLoaded = errors.New("refine materials from the SDE aren't loaded yet")

// Material is one thing a portion of an item refines into.
type Material struct {
	TypeID   int64
	Quantity int64
}

var (
	mu        sync.Mutex
	materials map[int64][]Material
	loadErr   error = ErrNotLoaded
)

// Materials returns what each type refines into, per portion, keyed by type
// ID. Types that can't be refined, such as gas, aren't in it.
func Materials() (map[int64][]Material, error) {
	mu.Lock()
	defer mu.Unlock()
	return materials, loadErr
}

// LoadInBackground reads the refine materials, downloading the SDE first
// when data/sde has no copy, retrying every hour until it works.
func LoadInBackground() {
	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			err := load(ctx)
			cancel()
			if err == nil {
				return
			}
			log.Printf("sde: %v", err)
			time.Sleep(time.Hour)
		}
	}()
}

func load(ctx context.Context) error {
	if _, err := os.Stat(TypeMaterialsPath); errors.Is(err, os.ErrNotExist) {
		url := os.Getenv("SDE_URL")
		if url == "" {
			url = defaultSdeURL
		}
		log.Printf("sde: %s is missing, downloading it from %s", TypeMaterialsPath, url)
		if err := download(ctx, url, TypeMaterialsPath); err != nil {
			setLoaded(nil, fmt.Errorf("downloading the SDE: %w", err))
			return err
		}
	}

	file, err := os.Open(TypeMaterialsPath)
	if err != nil {
		setLoaded(nil, err)
		return err
	}
	defer file.Close()
	loaded, err := ParseTypeMaterials(file)
	if err != nil {
		err = fmt.Errorf("reading %s: %w", TypeMaterialsPath, err)
	} else {
		log.Printf("sde: loaded refine materials for %d types", len(loaded))
	}
	setLoaded(loaded, err)
	return err
}

func setLoaded(loaded map[int64][]Material, err error) {
	mu.Lock()
	defer mu.Unlock()
	if err == nil {
		materials = loaded
	}
	if materials == nil || err == nil {
		loadErr = err
	}
}

// ParseTypeMaterials reads the SDE's typeMaterials.jsonl: one line per type,
// {"_key": typeID, "materials": [{"materialTypeID": ..., "quantity": ...}]}.
func ParseTypeMaterials(r io.Reader) (map[int64][]Material, error) {
	loaded := map[int64][]Material{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var entry struct {
			Key       int64 `json:"_key"`
			Materials []struct {
				MaterialTypeID int64 `json:"materialTypeID"`
				Quantity       int64 `json:"quantity"`
			} `json:"materials"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		for _, material := range entry.Materials {
			loaded[entry.Key] = append(loaded[entry.Key], Material{TypeID: material.MaterialTypeID, Quantity: material.Quantity})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(loaded) == 0 {
		return nil, errors.New("no materials found")
	}
	return loaded, nil
}

// download fetches the SDE zip and saves just its typeMaterials.jsonl to
// dest.
func download(ctx context.Context, url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	zipFile, err := os.CreateTemp(filepath.Dir(dest), "sde-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(zipFile.Name())
	defer zipFile.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %s", url, resp.Status)
	}
	size, err := io.Copy(zipFile, resp.Body)
	if err != nil {
		return err
	}

	archive, err := zip.NewReader(zipFile, size)
	if err != nil {
		return err
	}
	for _, entry := range archive.File {
		if path.Base(entry.Name) != filepath.Base(dest) {
			continue
		}
		return extract(entry, dest)
	}
	return fmt.Errorf("the SDE has no %s", filepath.Base(dest))
}

func extract(entry *zip.File, dest string) error {
	in, err := entry.Open()
	if err != nil {
		return err
	}
	defer in.Close()
	partial := dest + ".partial"
	out, err := os.Create(partial)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(partial, dest)
}
