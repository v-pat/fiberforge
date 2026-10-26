package blueprint

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	mu              sync.RWMutex
	embeddedCatalog = make(map[string]*Blueprint)
)

const (
	DefaultRegistryURL = "https://raw.githubusercontent.com/v-pat/fiberforge-blueprints/main/index.json"
	CacheTTLHours      = 12
)

// RegistryIndex represents the JSON catalog compiled in the remote GitHub registry.
type RegistryIndex struct {
	Version   string                      `json:"version"`
	UpdatedAt string                      `json:"updatedAt"`
	Items     map[string]BlueprintSummary `json:"blueprints"`
}

// Register registers a blueprint in the local/embedded in-memory catalog.
func Register(bp *Blueprint) {
	mu.Lock()
	defer mu.Unlock()
	embeddedCatalog[bp.ID] = bp
}

// Get retrieves a blueprint by ID. It searches embedded blueprints, then local cached blueprints.
func Get(id string) (*Blueprint, error) {
	mu.RLock()
	bp, ok := embeddedCatalog[id]
	mu.RUnlock()
	if ok {
		return bp, nil
	}

	// Check local cache directory ~/.fiberforge/cache/blueprints/<id>/
	home, err := os.UserHomeDir()
	if err == nil {
		cacheDir := filepath.Join(home, ".fiberforge", "cache", "blueprints", id)
		if info, err := os.Stat(cacheDir); err == nil && info.IsDir() {
			loadedBp, err := ParseBundle(cacheDir)
			if err == nil {
				return loadedBp, nil
			}
		}
	}

	return nil, fmt.Errorf("blueprint %q not found in embedded catalog or local cache", id)
}

// List returns a sorted slice of summaries of all registered and cached blueprints.
func List() []BlueprintSummary {
	mu.RLock()
	defer mu.RUnlock()

	var result []BlueprintSummary
	for _, bp := range embeddedCatalog {
		result = append(result, bp.Summary())
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result
}

// Search searches for blueprints matching query, category, and/or tag.
func Search(query string, category string, tag string) []BlueprintSummary {
	all := List()
	if query == "" && category == "" && tag == "" {
		return all
	}

	q := strings.ToLower(query)
	cat := strings.ToLower(category)
	tg := strings.ToLower(tag)

	var matches []BlueprintSummary
	for _, s := range all {
		if cat != "" && strings.ToLower(s.Category) != cat {
			continue
		}
		if tg != "" {
			hasTag := false
			for _, t := range s.Tags {
				if strings.ToLower(t) == tg {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue
			}
		}
		if q != "" {
			titleMatch := strings.Contains(strings.ToLower(s.Title), q)
			descMatch := strings.Contains(strings.ToLower(s.Description), q)
			idMatch := strings.Contains(strings.ToLower(s.ID), q)
			tagMatch := false
			for _, t := range s.Tags {
				if strings.Contains(strings.ToLower(t), q) {
					tagMatch = true
					break
				}
			}
			if !titleMatch && !descMatch && !idMatch && !tagMatch {
				continue
			}
		}
		matches = append(matches, s)
	}

	return matches
}

// FetchRemoteIndex retrieves the registry index from GitHub or cache.
func FetchRemoteIndex(forceRefresh bool) (*RegistryIndex, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	cacheFile := filepath.Join(home, ".fiberforge", "registry.json")

	if !forceRefresh {
		if info, err := os.Stat(cacheFile); err == nil {
			if time.Since(info.ModTime()) < CacheTTLHours*time.Hour {
				data, err := os.ReadFile(cacheFile)
				if err == nil {
					var idx RegistryIndex
					if err := json.Unmarshal(data, &idx); err == nil {
						return &idx, nil
					}
				}
			}
		}
	}

	// Fetch remote
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(DefaultRegistryURL)
	if err != nil {
		// Fallback to local cache if offline
		data, readErr := os.ReadFile(cacheFile)
		if readErr == nil {
			var idx RegistryIndex
			if err := json.Unmarshal(data, &idx); err == nil {
				return &idx, nil
			}
		}
		return nil, fmt.Errorf("fetching registry index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var idx RegistryIndex
	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		return nil, fmt.Errorf("decoding registry index: %w", err)
	}

	// Cache to disk
	_ = os.MkdirAll(filepath.Dir(cacheFile), 0o755)
	if out, err := json.Marshal(idx); err == nil {
		_ = os.WriteFile(cacheFile, out, 0o644)
	}

	return &idx, nil
}
