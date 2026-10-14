package esi

import "context"

type Asset struct {
	ItemId          int64  `json:"item_id"`
	TypeId          int64  `json:"type_id"`
	Quantity        int64  `json:"quantity"`
	LocationId      int64  `json:"location_id"`
	LocationFlag    string `json:"location_flag"` // e.g. Hangar, CorpSAG1, Cargo
	LocationType    string `json:"location_type"` // station, solar_system, item or other
	IsSingleton     bool   `json:"is_singleton"`
	IsBlueprintCopy bool   `json:"is_blueprint_copy"`
}

// GetCharacterAssets needs the esi-assets.read_assets.v1 scope.
func (c *Client) GetCharacterAssets(ctx context.Context, characterId int64, accessToken string) ([]Asset, error) {
	return getPages[Asset](ctx, c, c.url("/characters/%d/assets", characterId), accessToken)
}

// GetCorporationAssets needs the esi-assets.read_corporation_assets.v1
// scope and the Director role.
func (c *Client) GetCorporationAssets(ctx context.Context, corporationId int64, accessToken string) ([]Asset, error) {
	return getPages[Asset](ctx, c, c.url("/corporations/%d/assets", corporationId), accessToken)
}
