package refresh

import (
	"LifetimeIndustryTracker/src/db"
	"LifetimeIndustryTracker/src/sde"
	"context"
	"fmt"
	"log"
	"slices"
	"time"
)

// RecordDeliveries copies every delivered manufacturing and reaction job in
// industry_jobs into delivered_jobs, with how many items it made, then
// prices the items of every job that has no price yet.
func (r *Refresher) RecordDeliveries(ctx context.Context) error {
	if err := r.recordNewDeliveries(ctx); err != nil {
		return err
	}
	return r.priceDeliveries(ctx)
}

// recordNewDeliveries copies the jobs not yet in delivered_jobs. Jobs whose
// blueprint the SDE doesn't know are logged and tried again next time.
func (r *Refresher) recordNewDeliveries(ctx context.Context) error {
	jobs, err := db.LoadUnrecordedDeliveries(ctx)
	if err != nil || len(jobs) == 0 {
		return err
	}
	blueprints, err := sde.Blueprints()
	if err != nil {
		return err
	}

	ids := make([]int64, 0, 2*len(jobs))
	for _, job := range jobs {
		ids = append(ids, job.ProductTypeID, job.InstallerID)
	}
	names, err := r.ESI.GetNamesCached(ctx, ids)
	if err != nil {
		return fmt.Errorf("naming products and installers: %w", err)
	}

	recorded := make([]db.DeliveredJob, 0, len(jobs))
	for _, job := range jobs {
		products := blueprints[job.BlueprintTypeID].Manufacturing
		if job.ActivityID != db.ActivityManufacturing {
			products = blueprints[job.BlueprintTypeID].Reaction
		}
		unitsPerRun, ok := products[job.ProductTypeID]
		if !ok {
			log.Printf("refresh: delivered job %d: the SDE has no product %d for blueprint %d",
				job.JobID, job.ProductTypeID, job.BlueprintTypeID)
			continue
		}
		job.UnitsPerRun = unitsPerRun
		job.Quantity = job.Runs * unitsPerRun
		job.ProductName = names[job.ProductTypeID]
		job.InstallerName = names[job.InstallerID]
		recorded = append(recorded, job)
	}
	return db.SaveDeliveredJobs(ctx, recorded)
}

// priceDeliveries values each unpriced job's items at their Jita sell price
// on the day it was delivered, saving today's price of each first. A day
// with no saved price uses the closest day that has one, so jobs delivered
// before prices were saved take the earliest price. Items nobody sells in
// Jita stay unpriced and are tried again next time.
func (r *Refresher) priceDeliveries(ctx context.Context) error {
	jobs, err := db.LoadUnpricedDeliveries(ctx)
	if err != nil || len(jobs) == 0 {
		return err
	}
	var productIDs []int64
	for _, job := range jobs {
		if !slices.Contains(productIDs, job.ProductTypeID) {
			productIDs = append(productIDs, job.ProductTypeID)
		}
	}
	if err := r.SaveJitaPrices(ctx, productIDs, true); err != nil {
		log.Printf("refresh: today's Jita prices of delivered items: %v", err)
	}
	book, err := db.LoadJitaSellPrices(ctx, productIDs)
	if err != nil {
		return err
	}

	prices := map[int64]float64{}
	for _, job := range jobs {
		deliveredOn := job.CompletedDate.UTC().Truncate(24 * time.Hour)
		if price, ok := book.Closest(job.ProductTypeID, deliveredOn); ok {
			prices[job.JobID] = price
		}
	}
	return db.SetDeliveryPrices(ctx, prices)
}
