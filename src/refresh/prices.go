package refresh

import (
	"LifetimeIndustryTracker/src/db"
	"LifetimeIndustryTracker/src/mining"
	"LifetimeIndustryTracker/src/sde"
	"context"
	"fmt"
	"slices"
	"sync"
)

// base ore type IDs never change, so each name is looked up once per run
var (
	baseOreMu  sync.Mutex
	baseOreIDs = map[string]int64{}
)

// OreRecipes works out what one unit of each ore refines into, keyed by the
// ore's type ID: its base ore's refine materials from the SDE, with the
// grade bonus added. Ores whose base can't be refined, such as gas, are left
// out.
func (r *Refresher) OreRecipes(ctx context.Context, oreTypes map[int64]string) (map[int64]mining.Recipe, error) {
	materials, err := sde.Materials()
	if err != nil {
		return nil, err
	}

	var unknown []string
	baseOreMu.Lock()
	for _, name := range oreTypes {
		base := mining.BaseOreName(name)
		if _, ok := baseOreIDs[base]; !ok && !slices.Contains(unknown, base) {
			unknown = append(unknown, base)
		}
	}
	baseOreMu.Unlock()
	if len(unknown) > 0 {
		found, err := r.ESI.GetTypeIds(ctx, unknown)
		if err != nil {
			return nil, fmt.Errorf("looking up base ores: %w", err)
		}
		baseOreMu.Lock()
		for _, base := range unknown {
			baseOreIDs[base] = found[base] // 0 when ESI doesn't know it
		}
		baseOreMu.Unlock()
	}

	recipes := make(map[int64]mining.Recipe, len(oreTypes))
	for typeID, name := range oreTypes {
		baseOreMu.Lock()
		baseID := baseOreIDs[mining.BaseOreName(name)]
		baseOreMu.Unlock()
		baseMaterials := materials[baseID]
		if baseID == 0 || len(baseMaterials) == 0 {
			continue
		}
		portionSize, err := r.ESI.GetPortionSize(ctx, baseID)
		if err != nil {
			return nil, fmt.Errorf("portion size of %s: %w", mining.BaseOreName(name), err)
		}
		quantities := make(map[int64]int64, len(baseMaterials))
		for _, material := range baseMaterials {
			quantities[material.TypeID] += material.Quantity
		}
		recipes[typeID] = mining.NewRecipe(name, quantities, portionSize)
	}
	return recipes, nil
}

// MaterialIDs returns every material the recipes refine into.
func MaterialIDs(recipes map[int64]mining.Recipe) []int64 {
	var ids []int64
	for _, recipe := range recipes {
		for typeID := range recipe {
			if !slices.Contains(ids, typeID) {
				ids = append(ids, typeID)
			}
		}
	}
	slices.Sort(ids)
	return ids
}

// SaveJitaPrices saves today's Jita sell price of each material. With
// onlyMissing it skips materials already priced today, so a page load only
// pays for the first fetch of the day; the hourly refresh updates them all.
// A material nobody sells in Jita gets no price.
func (r *Refresher) SaveJitaPrices(ctx context.Context, typeIDs []int64, onlyMissing bool) error {
	date := today()
	priced := map[int64]bool{}
	if onlyMissing {
		var err error
		if priced, err = db.PricedOn(ctx, typeIDs, date); err != nil {
			return err
		}
	}
	// prices fetched before an error are still saved
	prices := map[int64]float64{}
	var fetchErr error
	for _, typeID := range typeIDs {
		if priced[typeID] {
			continue
		}
		price, found, err := r.ESI.GetJitaSellPrice(ctx, typeID)
		if err != nil {
			fetchErr = fmt.Errorf("Jita sell price of %d: %w", typeID, err)
			break
		}
		if found {
			prices[typeID] = price
		}
	}
	if err := db.SaveJitaSellPrices(ctx, date, prices); err != nil {
		return err
	}
	return fetchErr
}

// saveOrePrices saves today's Jita sell price of everything any mined ore
// refines into.
func (r *Refresher) saveOrePrices(ctx context.Context) error {
	oreTypes, err := db.LoadMinedOreTypes(ctx)
	if err != nil {
		return err
	}
	recipes, err := r.OreRecipes(ctx, oreTypes)
	if err != nil {
		return err
	}
	return r.SaveJitaPrices(ctx, MaterialIDs(recipes), false)
}
