package esi

import "context"

// MiningLedgerEntry is how much of one ore a character mined in one solar
// system on one day.
type MiningLedgerEntry struct {
	Date          string `json:"date"` // YYYY-MM-DD
	SolarSystemId int64  `json:"solar_system_id"`
	TypeId        int64  `json:"type_id"`
	Quantity      int64  `json:"quantity"`
}

// MiningObserver is a corporation structure, such as a moon drill
// refinery, that records what is mined around it.
type MiningObserver struct {
	ObserverId   int64  `json:"observer_id"`
	ObserverType string `json:"observer_type"` // structure
	LastUpdated  string `json:"last_updated"`  // YYYY-MM-DD
}

// ObservedMiningEntry is how much of one ore a character mined at an
// observer, as of LastUpdated.
type ObservedMiningEntry struct {
	CharacterId           int64  `json:"character_id"`
	RecordedCorporationId int64  `json:"recorded_corporation_id"`
	TypeId                int64  `json:"type_id"`
	Quantity              int64  `json:"quantity"`
	LastUpdated           string `json:"last_updated"` // YYYY-MM-DD
}

// CorporationMiningLedger is everything a corporation's observers recorded.
type CorporationMiningLedger struct {
	Observer MiningObserver
	Entries  []ObservedMiningEntry
}

// GetCharacterMiningLedger returns the last 30 days of the character's
// mining. Needs the esi-industry.read_character_mining.v1 scope.
func (c *Client) GetCharacterMiningLedger(ctx context.Context, characterId int64, accessToken string) ([]MiningLedgerEntry, error) {
	return getPages[MiningLedgerEntry](ctx, c, c.url("/characters/%d/mining", characterId), accessToken)
}

// GetCorporationMiningObservers lists the corporation's observers. Needs the
// esi-industry.read_corporation_mining.v1 scope and the Accountant role.
// ESI spells this route /corporation/, not /corporations/.
func (c *Client) GetCorporationMiningObservers(ctx context.Context, corporationId int64, accessToken string) ([]MiningObserver, error) {
	return getPages[MiningObserver](ctx, c, c.url("/corporation/%d/mining/observers", corporationId), accessToken)
}

// GetCorporationObservedMining returns the last 30 days of what was mined at
// one observer. Same scope and role as GetCorporationMiningObservers.
func (c *Client) GetCorporationObservedMining(ctx context.Context, corporationId, observerId int64, accessToken string) ([]ObservedMiningEntry, error) {
	return getPages[ObservedMiningEntry](ctx, c, c.url("/corporation/%d/mining/observers/%d", corporationId, observerId), accessToken)
}

// GetCorporationMiningLedger is the corporation's whole mining ledger: every
// observer and what was mined at each.
func (c *Client) GetCorporationMiningLedger(ctx context.Context, corporationId int64, accessToken string) ([]CorporationMiningLedger, error) {
	observers, err := c.GetCorporationMiningObservers(ctx, corporationId, accessToken)
	if err != nil {
		return nil, err
	}
	ledgers := make([]CorporationMiningLedger, 0, len(observers))
	for _, observer := range observers {
		entries, err := c.GetCorporationObservedMining(ctx, corporationId, observer.ObserverId, accessToken)
		if err != nil {
			return nil, err
		}
		ledgers = append(ledgers, CorporationMiningLedger{Observer: observer, Entries: entries})
	}
	return ledgers, nil
}
