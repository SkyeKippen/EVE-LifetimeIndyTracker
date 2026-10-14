package esi

import (
	"context"
	"sync"
)

// GetNames looks up the names of characters, corporations, systems, types
// and other entities by ID, 1000 at a time. No scope needed.
func (c *Client) GetNames(ctx context.Context, ids []int64) (map[int64]string, error) {
	names := make(map[int64]string, len(ids))
	unique := make([]int64, 0, len(ids))
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	for start := 0; start < len(unique); start += 1000 {
		chunk := unique[start:min(start+1000, len(unique))]
		var resolved []struct {
			Id   int64  `json:"id"`
			Name string `json:"name"`
		}
		if _, err := c.doWithRetry(ctx, "POST", c.url("/universe/names"), "", chunk, &resolved); err != nil {
			return nil, err
		}
		for _, entry := range resolved {
			names[entry.Id] = entry.Name
		}
	}
	return names, nil
}

// type and solar system names never change, so they are looked up once per
// server run
var (
	namesMu     sync.Mutex
	cachedNames = map[int64]string{}
)

// GetNamesCached is GetNames, remembering every answer. Only use it for
// things whose names never change, such as types and solar systems.
func (c *Client) GetNamesCached(ctx context.Context, ids []int64) (map[int64]string, error) {
	names := make(map[int64]string, len(ids))
	var missing []int64
	namesMu.Lock()
	for _, id := range ids {
		if name, ok := cachedNames[id]; ok {
			names[id] = name
		} else {
			missing = append(missing, id)
		}
	}
	namesMu.Unlock()
	if len(missing) == 0 {
		return names, nil
	}

	resolved, err := c.GetNames(ctx, missing)
	if err != nil {
		return nil, err
	}
	namesMu.Lock()
	for id, name := range resolved {
		cachedNames[id] = name
		names[id] = name
	}
	namesMu.Unlock()
	return names, nil
}
