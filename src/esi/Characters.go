package esi

import "context"

type Character struct {
	Name          string `json:"name"`
	CorporationId int64  `json:"corporation_id"`
	AllianceId    int64  `json:"alliance_id"`
}

// GetCharacter returns a character's public info, including the corporation
// whose wallets, orders and assets the corporation queries need. No scope
// needed.
func (c *Client) GetCharacter(ctx context.Context, characterId int64) (Character, error) {
	return get[Character](ctx, c, c.url("/characters/%d", characterId), "")
}
