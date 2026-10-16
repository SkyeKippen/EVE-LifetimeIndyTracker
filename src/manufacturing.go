package main

import (
	"LifetimeIndustryTracker/src/auth"
	"LifetimeIndustryTracker/src/db"
	"LifetimeIndustryTracker/src/production"
	"LifetimeIndustryTracker/src/refresh"
	"LifetimeIndustryTracker/src/sde"
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

// ManufacturingColumn is one sortable heading of the Manufacturing table.
type ManufacturingColumn struct {
	Label   string
	Unit    string // shown after the label, e.g. ISK
	Numeric bool
	Href    string // the link that sorts by this column
	Arrow   string // ▲ or ▼ on the column the list is sorted by
}

func renderManufacturing(w http.ResponseWriter, r *http.Request) {
	sess, _ := auth.CurrentSession(r)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// save the latest jobs first so the totals include today's deliveries
	userChars, err := userCharacters(ctx, sess)
	if err != nil {
		log.Printf("renderManufacturing: characters of user %d: %v", sess.UserID, err)
		http.Error(w, "could not load your characters", http.StatusInternalServerError)
		return
	}
	var characterIDs []int64
	var characters []MiningCharacter
	for _, character := range userChars {
		characterIDs = append(characterIDs, character.CharacterID)
		problem := ""
		if err := refresher.CharacterJobs(ctx, character); err != nil {
			log.Printf("renderManufacturing: industry jobs of %d (%s): %v", character.CharacterID, character.CharacterName, err)
			problem = jobsProblem(err)
		}
		characters = append(characters, MiningCharacter{CharacterName: character.CharacterName, Problem: problem})
	}
	recordProblem := ""
	if err := refresher.RecordDeliveries(ctx); errors.Is(err, sde.ErrNotLoaded) {
		recordProblem = "EVE's static data is still loading, so check back in a few minutes"
	} else if err != nil {
		log.Printf("renderManufacturing: recording deliveries: %v", err)
		recordProblem = "they couldn't be added up"
	}

	// reactions count too, each product (intermediates included) as its own
	// item; only jobs the user's own characters started count
	totals, err := db.LoadLifetimeProduction(ctx, characterIDs,
		[]int{db.ActivityManufacturing, db.ActivityOldReaction, db.ActivityReaction})
	if err != nil {
		log.Printf("renderManufacturing: loading lifetime production: %v", err)
		http.Error(w, "could not load saved manufacturing: "+err.Error(), http.StatusInternalServerError)
		return
	}
	items := make([]production.Item, 0, len(totals))
	var grandQuantity, unpriced int64
	var grandValue float64
	for _, total := range totals {
		category, group, _ := sde.ItemType(total.ProductTypeID)
		items = append(items, production.Item{
			Name: total.ProductName, Category: category, Group: group,
			Quantity: total.Quantity, Value: total.Value, UnpricedQuantity: total.UnpricedQuantity,
		})
		grandQuantity += total.Quantity
		grandValue += total.Value
		unpriced += total.UnpricedQuantity
	}
	sort := production.ParseSort(r.URL.Query().Get("sort"), r.URL.Query().Get("dir"))
	production.SortItems(items, sort)

	data := struct {
		CharacterName string
		Characters    []MiningCharacter
		RecordProblem string
		Columns       []ManufacturingColumn
		Items         []production.Item
		GrandQuantity int64
		GrandValue    float64
		Unpriced      int64
	}{CharacterName: sess.CharacterName, Characters: characters, RecordProblem: recordProblem,
		Columns: manufacturingColumns(sort), Items: items,
		GrandQuantity: grandQuantity, GrandValue: grandValue, Unpriced: unpriced}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "manufacturing.html", data); err != nil {
		log.Printf("renderManufacturing: %v", err)
	}
}

// manufacturingColumns gives each heading the link that sorts by it.
func manufacturingColumns(current production.Sort) []ManufacturingColumn {
	columns := []ManufacturingColumn{
		{Label: "Item", Href: production.SortName},
		{Label: "Item Type", Href: production.SortType},
		{Label: "Quantity Made", Unit: "units", Numeric: true, Href: production.SortQuantity},
		{Label: "Total Value", Unit: "ISK", Numeric: true, Href: production.SortValue},
	}
	for i, column := range columns {
		next := current.Toggle(column.Href)
		if current.Column == column.Href {
			columns[i].Arrow = "▲"
			if current.Desc {
				columns[i].Arrow = "▼"
			}
		}
		columns[i].Href = "/manufacturing?sort=" + next.Column + "&dir=" + next.Dir()
	}
	return columns
}

// jobsProblem says, for the page, why a character's latest jobs are
// missing.
func jobsProblem(err error) string {
	switch {
	case errors.Is(err, refresh.ErrMissingScope):
		return "didn't grant Character industry jobs at sign-in"
	case errors.Is(err, db.ErrNoToken):
		return "needs to sign in again"
	case errors.Is(err, db.ErrNoDatabase):
		return "the database isn't set up"
	default:
		return "their latest jobs couldn't be fetched or saved"
	}
}
