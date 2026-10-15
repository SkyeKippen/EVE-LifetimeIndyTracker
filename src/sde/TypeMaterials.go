// Package sde reads what each item refines into and each item's volume from
// EVE's static data export (SDE). ESI has no route for refine materials, so
// the app keeps the SDE's typeMaterials.jsonl and types.jsonl in data/sde,
// downloading them from CCP the first time either is missing.
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
	"slices"
	"sync"
	"time"
)

// Where the SDE files are kept, relative to the working directory like
// config/.env.
const (
	TypeMaterialsPath = "data/sde/typeMaterials.jsonl"
	TypesPath         = "data/sde/types.jsonl"
)

// defaultSdeURL is CCP's latest SDE in JSON Lines; SDE_URL in config/.env
// overrides it.
const defaultSdeURL = "https://developers.eveonline.com/static-data/eve-online-static-data-latest-jsonl.zip"

// ErrNotLoaded means the SDE hasn't been read yet, e.g. while it's still
// downloading.
var ErrNotLoaded = errors.New("EVE's static data (SDE) isn't loaded yet")

// Material is one thing a portion of an item refines into.
type Material struct {
	TypeID   int64
	Quantity int64
}

var (
	mu        sync.Mutex
	materials map[int64][]Material
	volumes   map[int64]float64
	loadErr   error = ErrNotLoaded
)

// Materials returns what each type refines into, per portion, keyed by type
// ID. Types that can't be refined, such as gas, aren't in it.
func Materials() (map[int64][]Material, error) {
	mu.Lock()
	defer mu.Unlock()
	return materials, loadErr
}

// Volumes returns the volume of one unit of each type in m3, keyed by type
// ID.
func Volumes() (map[int64]float64, error) {
	mu.Lock()
	defer mu.Unlock()
	return volumes, loadErr
}

// LoadInBackground reads the SDE files, downloading the SDE first when
// data/sde is missing any of them, retrying every hour until it works.
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
	var missing []string
	for _, dest := range []string{TypeMaterialsPath, TypesPath} {
		if _, err := os.Stat(dest); errors.Is(err, os.ErrNotExist) {
			missing = append(missing, dest)
		}
	}
	if len(missing) > 0 {
		url := os.Getenv("SDE_URL")
		if url == "" {
			url = defaultSdeURL
		}
		log.Printf("sde: %v missing, downloading the SDE from %s", missing, url)
		if err := download(ctx, url, missing); err != nil {
			err = fmt.Errorf("downloading the SDE: %w", err)
			setLoaded(nil, nil, err)
			return err
		}
	}

	loadedMaterials, err := parseFile(TypeMaterialsPath, ParseTypeMaterials)
	if err != nil {
		setLoaded(nil, nil, err)
		return err
	}
	loadedVolumes, err := parseFile(TypesPath, ParseVolumes)
	if err != nil {
		setLoaded(nil, nil, err)
		return err
	}
	log.Printf("sde: loaded refine materials for %d types and volumes for %d", len(loadedMaterials), len(loadedVolumes))
	setLoaded(loadedMaterials, loadedVolumes, nil)
	return nil
}

func parseFile[T any](name string, parse func(io.Reader) (T, error)) (T, error) {
	file, err := os.Open(name)
	if err != nil {
		var zero T
		return zero, err
	}
	defer file.Close()
	parsed, err := parse(file)
	if err != nil {
		return parsed, fmt.Errorf("reading %s: %w", name, err)
	}
	return parsed, nil
}

// setLoaded keeps what was loaded before when a later load fails.
func setLoaded(loadedMaterials map[int64][]Material, loadedVolumes map[int64]float64, err error) {
	mu.Lock()
	defer mu.Unlock()
	if err == nil {
		materials, volumes = loadedMaterials, loadedVolumes
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

// ParseVolumes reads each type's volume from the SDE's types.jsonl: one
// line per type, {"_key": typeID, "volume": m3, ...}.
func ParseVolumes(r io.Reader) (map[int64]float64, error) {
	loaded := map[int64]float64{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var entry struct {
			Key    int64    `json:"_key"`
			Volume *float64 `json:"volume"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		if entry.Volume != nil {
			loaded[entry.Key] = *entry.Volume
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(loaded) == 0 {
		return nil, errors.New("no volumes found")
	}
	return loaded, nil
}

// download fetches the SDE zip and saves just the files named by dests,
// matching each by its base name.
func download(ctx context.Context, url string, dests []string) error {
	if err := os.MkdirAll(filepath.Dir(dests[0]), 0o755); err != nil {
		return err
	}
	zipFile, err := os.CreateTemp(filepath.Dir(dests[0]), "sde-*.zip")
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
	for _, dest := range dests {
		i := slices.IndexFunc(archive.File, func(entry *zip.File) bool {
			return path.Base(entry.Name) == filepath.Base(dest)
		})
		if i < 0 {
			return fmt.Errorf("the SDE has no %s", filepath.Base(dest))
		}
		if err := extract(archive.File[i], dest); err != nil {
			return err
		}
	}
	return nil
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
