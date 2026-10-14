package esi

import (
	"context"
	"errors"
	"log"
)

type Structure struct {
	StructureId   int64  `json:"-"`
	Name          string `json:"name"`
	OwnerId       int64  `json:"owner_id"` // corporation
	SolarSystemId int64  `json:"solar_system_id"`
	TypeId        int64  `json:"type_id"`
}

// IsStructureId reports whether a location ID is a player-owned structure
// (Upwell structures have IDs over a trillion; NPC stations are 60-64
// million).
func IsStructureId(locationId int64) bool {
	return locationId > 1_000_000_000_000
}

// GetStructure needs the esi-universe.read_structures.v1 scope, and the
// character must be on the structure's access list or it returns
// ErrForbidden.
func (c *Client) GetStructure(ctx context.Context, structureId int64, accessToken string) (Structure, error) {
	structure, err := get[Structure](ctx, c, c.url("/universe/structures/%d", structureId), accessToken)
	structure.StructureId = structureId
	return structure, err
}

// GetPublicStructureIds lists every structure open to everyone. No scope
// needed.
func (c *Client) GetPublicStructureIds(ctx context.Context) ([]int64, error) {
	return get[[]int64](ctx, c, c.url("/universe/structures"), "")
}

// GetAccessibleStructures returns the structures among candidateIds that the
// token's character can see, skipping the ones it can't. ESI has no route
// that lists every structure a character can use, so the candidates come
// from what the character or corporation already has: the location IDs of
// its assets, market orders and industry jobs. Each refusal counts toward
// ESI's error limit, so don't pass in thousands of unrelated structures.
func (c *Client) GetAccessibleStructures(ctx context.Context, candidateIds []int64, accessToken string) ([]Structure, error) {
	seen := make(map[int64]bool, len(candidateIds))
	structures := make([]Structure, 0)
	for _, id := range candidateIds {
		if seen[id] || !IsStructureId(id) {
			continue
		}
		seen[id] = true

		structure, err := c.GetStructure(ctx, id, accessToken)
		if errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotFound) {
			log.Printf("esi: structure %d isn't accessible: %v", id, err)
			continue
		}
		if err != nil {
			return nil, err
		}
		structures = append(structures, structure)
	}
	return structures, nil
}

// StructureIdsOf collects the structure IDs that assets, market orders and
// industry jobs sit in, as candidates for GetAccessibleStructures.
func StructureIdsOf(assets []Asset, orders []MarketOrder, jobs []IndustryJob) []int64 {
	var ids []int64
	add := func(id int64) {
		if IsStructureId(id) {
			ids = append(ids, id)
		}
	}
	// an asset inside a container, ship or office has that item as its
	// location, and item IDs look like structure IDs, so only take locations
	// that aren't themselves one of the assets
	owned := make(map[int64]bool, len(assets))
	for _, asset := range assets {
		owned[asset.ItemId] = true
	}
	for _, asset := range assets {
		if !owned[asset.LocationId] {
			add(asset.LocationId)
		}
	}
	for _, order := range orders {
		add(order.LocationId)
	}
	for _, job := range jobs {
		add(job.LocationId)
		add(job.FacilityId)
	}
	return ids
}
