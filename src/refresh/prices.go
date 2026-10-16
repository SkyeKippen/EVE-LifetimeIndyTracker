package refresh

import (
	"LifetimeIndustryTracker/src/db"
	"LifetimeIndustryTracker/src/mining"
	"LifetimeIndustryTracker/src/sde"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"
)

// OreRecipes works out what one unit of each ore refines into, keyed by the
// ore's type ID: its base ore's refine materials from the SDE, with the
// grade bonus added. Ores whose base can't be refined, such as gas, are left
// out. It reads only the SDE.
func OreRecipes(oreTypes map[int64]string) (map[int64]mining.Recipe, error) {
	materials, err := sde.Materials()
	if err != nil {
		return nil, err
	}
	recipes := make(map[int64]mining.Recipe, len(oreTypes))
	for typeID, name := range oreTypes {
		baseID, ok := sde.TypeIDByName(mining.BaseOreName(name))
		if !ok || len(materials[baseID]) == 0 {
			continue
		}
		quantities := make(map[int64]int64, len(materials[baseID]))
		for _, material := range materials[baseID] {
			quantities[material.TypeID] += material.Quantity
		}
		recipes[typeID] = mining.NewRecipe(name, quantities, sde.PortionSize(baseID))
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

// types nobody sells in Jita get no saved price, so the day they were last
// looked up is kept here to look them up only once a day too
var (
	unsoldMu  sync.Mutex
	unsoldDay = map[int64]time.Time{}
)

// SaveJitaPrices saves today's Jita sell price of each type that has none
// yet, so each is fetched from ESI at most once a day.
func (r *Refresher) SaveJitaPrices(ctx context.Context, typeIDs []int64) error {
	date := today()
	priced, err := db.PricedOn(ctx, typeIDs, date)
	if err != nil {
		return err
	}
	// prices fetched before an error are still saved
	prices := map[int64]float64{}
	var fetchErr error
	for _, typeID := range typeIDs {
		unsoldMu.Lock()
		checked := unsoldDay[typeID].Equal(date)
		unsoldMu.Unlock()
		if priced[typeID] || checked {
			continue
		}
		price, found, err := r.ESI.GetJitaSellPrice(ctx, typeID)
		if err != nil {
			fetchErr = fmt.Errorf("Jita sell price of %d: %w", typeID, err)
			break
		}
		if found {
			prices[typeID] = price
		} else {
			unsoldMu.Lock()
			unsoldDay[typeID] = date
			unsoldMu.Unlock()
		}
	}
	if err := db.SaveJitaSellPrices(ctx, date, prices); err != nil {
		return err
	}
	return fetchErr
}

// saveDailyPrices saves today's Jita sell price of everything the pages
// value: what any mined ore refines into and every item ever delivered.
func (r *Refresher) saveDailyPrices(ctx context.Context) error {
	oreTypes, err := db.LoadMinedOreTypes(ctx)
	if err != nil {
		return err
	}
	recipes, err := OreRecipes(oreTypes)
	if err != nil {
		return err
	}
	productIDs, err := db.LoadDeliveredProductIDs(ctx)
	if err != nil {
		return err
	}
	typeIDs := MaterialIDs(recipes)
	for _, productID := range productIDs {
		if !slices.Contains(typeIDs, productID) {
			typeIDs = append(typeIDs, productID)
		}
	}
	return r.SaveJitaPrices(ctx, typeIDs)
}
