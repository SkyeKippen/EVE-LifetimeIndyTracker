package esi

import "context"

// Jita is where ore values are priced: The Forge region and the Jita IV -
// Moon 4 - Caldari Navy Assembly Plant station.
const (
	TheForgeRegionId = 10000002
	JitaStationId    = 60003760
)

// GetJitaSellPrice returns the lowest sell order for the type at the Jita
// 4-4 station, and false when nobody is selling it there. No scope needed.
func (c *Client) GetJitaSellPrice(ctx context.Context, typeId int64) (float64, bool, error) {
	orders, err := getPages[struct {
		LocationId int64   `json:"location_id"`
		Price      float64 `json:"price"`
	}](ctx, c, c.url("/markets/%d/orders?order_type=sell&type_id=%d", TheForgeRegionId, typeId), "")
	if err != nil {
		return 0, false, err
	}
	lowest, found := 0.0, false
	for _, order := range orders {
		if order.LocationId == JitaStationId && (!found || order.Price < lowest) {
			lowest, found = order.Price, true
		}
	}
	return lowest, found, nil
}
