package esi

import "context"

// LoyaltyPoints is how many LP a character has with one NPC corporation.
type LoyaltyPoints struct {
	CorporationId int64 `json:"corporation_id"`
	LoyaltyPoints int64 `json:"loyalty_points"`
}

// GetCharacterLoyaltyPoints needs the esi-characters.read_loyalty.v1 scope.
// ESI has no route for a corporation's LP, since only characters hold LP.
func (c *Client) GetCharacterLoyaltyPoints(ctx context.Context, characterId int64, accessToken string) ([]LoyaltyPoints, error) {
	return get[[]LoyaltyPoints](ctx, c, c.url("/characters/%d/loyalty/points", characterId), accessToken)
}
