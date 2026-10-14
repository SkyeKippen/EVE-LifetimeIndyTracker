package main

import (
	"LifetimeIndustryTracker/src/auth"
	"LifetimeIndustryTracker/src/db"
	"LifetimeIndustryTracker/src/esi"
	"LifetimeIndustryTracker/src/mining"
	"LifetimeIndustryTracker/src/refresh"
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

// esiClient makes every ESI call the pages need.
var esiClient = &esi.Client{}

// refresher saves ESI data to the database, hourly and on demand.
var refresher = &refresh.Refresher{ESI: esiClient}

// MiningCharacter is a character whose mining ledger the page tried to read.
type MiningCharacter struct {
	CharacterName string
	Problem       string // why their latest mining couldn't be saved, or ""
}

// userCharacters returns every character of the signed-in user, main and
// alts, with the scopes each granted. A character whose token is gone is
// still included so the page can say it needs to sign in again.
func userCharacters(ctx context.Context, sess *auth.Session) ([]db.TokenCharacter, error) {
	linked, err := db.ListUserCharacters(ctx, sess.UserID)
	if err != nil {
		return nil, err
	}
	characters := make([]db.TokenCharacter, 0, len(linked))
	for _, character := range linked {
		withToken, err := db.LoadTokenCharacter(ctx, character.CharacterID)
		if err != nil {
			withToken = db.TokenCharacter{CharacterID: character.CharacterID, CharacterName: character.CharacterName}
		}
		characters = append(characters, withToken)
	}
	return characters, nil
}

func renderMining(w http.ResponseWriter, r *http.Request) {
	sess, _ := auth.CurrentSession(r)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// save the latest mining first so the totals include today
	var characterIDs []int64
	var characters []MiningCharacter
	userChars, err := userCharacters(ctx, sess)
	if err != nil {
		log.Printf("renderMining: characters of user %d: %v", sess.UserID, err)
		http.Error(w, "could not load your characters", http.StatusInternalServerError)
		return
	}
	for _, character := range userChars {
		characterIDs = append(characterIDs, character.CharacterID)
		problem := ""
		err := refresher.CharacterMining(ctx, character)
		if err != nil {
			log.Printf("renderMining: mining ledger of %d (%s): %v", character.CharacterID, character.CharacterName, err)
			problem = miningProblem(err)
		}
		characters = append(characters, MiningCharacter{CharacterName: character.CharacterName, Problem: problem})
	}

	ores, err := db.LoadOreTotals(ctx, characterIDs)
	if err != nil {
		log.Printf("renderMining: loading ore totals: %v", err)
		http.Error(w, "could not load saved mining: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var grandTotal int64
	for _, ore := range ores {
		grandTotal += ore.Quantity
	}

	data := struct {
		CharacterName string
		Characters    []MiningCharacter
		Ores          []mining.OreTotal
		GrandTotal    int64
	}{CharacterName: sess.CharacterName, Characters: characters, Ores: ores, GrandTotal: grandTotal}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.ExecuteTemplate(w, "mining.html", data)
	if err != nil {
		log.Printf("renderMining: %v", err)
	}
}

// miningProblem says, for the page, why a character's latest mining is
// missing.
func miningProblem(err error) string {
	switch {
	case errors.Is(err, refresh.ErrMissingScope):
		return "didn't grant Character mining ledger at sign-in"
	case errors.Is(err, db.ErrNoToken):
		return "needs to sign in again"
	case errors.Is(err, db.ErrNoDatabase):
		return "the database isn't set up"
	case errors.Is(err, esi.ErrForbidden):
		return "ESI refused access to the mining ledger"
	default:
		return "their latest mining couldn't be fetched or saved"
	}
}
