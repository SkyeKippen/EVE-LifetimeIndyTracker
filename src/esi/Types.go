package esi

import (
	"context"
	"sync"
)

// GetTypeIds looks up item types by their exact English name. Names ESI
// doesn't know are left out. No scope needed.
func (c *Client) GetTypeIds(ctx context.Context, names []string) (map[string]int64, error) {
	ids := make(map[string]int64, len(names))
	for start := 0; start < len(names); start += 500 {
		chunk := names[start:min(start+500, len(names))]
		var resolved struct {
			InventoryTypes []struct {
				Id   int64  `json:"id"`
				Name string `json:"name"`
			} `json:"inventory_types"`
		}
		if _, err := c.doWithRetry(ctx, "POST", c.url("/universe/ids"), "", chunk, &resolved); err != nil {
			return nil, err
		}
		for _, itemType := range resolved.InventoryTypes {
			ids[itemType.Name] = itemType.Id
		}
	}
	return ids, nil
}

// portion sizes never change, so each is looked up once per server run
var (
	portionMu      sync.Mutex
	cachedPortions = map[int64]int64{}
)

// GetPortionSize returns how many units of the type are refined at a time,
// e.g. 100 for asteroid ore and 1 for ice. No scope needed.
func (c *Client) GetPortionSize(ctx context.Context, typeId int64) (int64, error) {
	portionMu.Lock()
	size, ok := cachedPortions[typeId]
	portionMu.Unlock()
	if ok {
		return size, nil
	}

	itemType, err := get[struct {
		PortionSize int64 `json:"portion_size"`
	}](ctx, c, c.url("/universe/types/%d", typeId), "")
	if err != nil {
		return 0, err
	}
	size = max(itemType.PortionSize, 1)
	portionMu.Lock()
	cachedPortions[typeId] = size
	portionMu.Unlock()
	return size, nil
}
