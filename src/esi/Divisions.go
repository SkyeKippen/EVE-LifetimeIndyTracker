package esi

import "context"

// Division is one of a corporation's seven hangar or wallet divisions. Name
// is empty while the division still has its default name.
type Division struct {
	Division int    `json:"division"`
	Name     string `json:"name"`
}

type CorporationDivisions struct {
	Hangar []Division `json:"hangar"`
	Wallet []Division `json:"wallet"`
}

// GetCorporationDivisions needs the esi-corporations.read_divisions.v1 scope
// and the Director role.
func (c *Client) GetCorporationDivisions(ctx context.Context, corporationId int64, accessToken string) (CorporationDivisions, error) {
	return get[CorporationDivisions](ctx, c, c.url("/corporations/%d/divisions", corporationId), accessToken)
}
