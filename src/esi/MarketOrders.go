package esi

import (
	"context"
	"time"
)

// MarketOrder is an open or past market order of a character or
// corporation. Some fields only come with one or the other: IsCorporation
// for characters, IssuedBy and WalletDivision for corporations, and State
// only on order history.
type MarketOrder struct {
	OrderId        int64     `json:"order_id"`
	TypeId         int64     `json:"type_id"`
	IsBuyOrder     bool      `json:"is_buy_order"`
	Price          float64   `json:"price"`
	VolumeTotal    int64     `json:"volume_total"`
	VolumeRemain   int64     `json:"volume_remain"`
	MinVolume      int64     `json:"min_volume"`
	Escrow         float64   `json:"escrow"`
	Range          string    `json:"range"`
	Duration       int       `json:"duration"` // days
	Issued         time.Time `json:"issued"`
	LocationId     int64     `json:"location_id"`
	RegionId       int64     `json:"region_id"`
	IsCorporation  bool      `json:"is_corporation"`
	IssuedBy       int64     `json:"issued_by"`
	WalletDivision int       `json:"wallet_division"`
	State          string    `json:"state"` // cancelled or expired (expired includes filled)
}

// GetCharacterMarketOrders returns the character's open orders. Needs the
// esi-markets.read_character_orders.v1 scope.
func (c *Client) GetCharacterMarketOrders(ctx context.Context, characterId int64, accessToken string) ([]MarketOrder, error) {
	return get[[]MarketOrder](ctx, c, c.url("/characters/%d/orders", characterId), accessToken)
}

// GetCharacterMarketOrderHistory returns the character's cancelled and
// expired orders from the last 90 days. Needs the
// esi-markets.read_character_orders.v1 scope.
func (c *Client) GetCharacterMarketOrderHistory(ctx context.Context, characterId int64, accessToken string) ([]MarketOrder, error) {
	return getPages[MarketOrder](ctx, c, c.url("/characters/%d/orders/history", characterId), accessToken)
}

// GetCorporationMarketOrders returns the corporation's open orders. Needs the
// esi-markets.read_corporation_orders.v1 scope and the Accountant or Trader
// role.
func (c *Client) GetCorporationMarketOrders(ctx context.Context, corporationId int64, accessToken string) ([]MarketOrder, error) {
	return getPages[MarketOrder](ctx, c, c.url("/corporations/%d/orders", corporationId), accessToken)
}

// GetCorporationMarketOrderHistory returns the corporation's cancelled and
// expired orders from the last 90 days. Needs the
// esi-markets.read_corporation_orders.v1 scope and the Accountant or Trader
// role.
func (c *Client) GetCorporationMarketOrderHistory(ctx context.Context, corporationId int64, accessToken string) ([]MarketOrder, error) {
	return getPages[MarketOrder](ctx, c, c.url("/corporations/%d/orders/history", corporationId), accessToken)
}
