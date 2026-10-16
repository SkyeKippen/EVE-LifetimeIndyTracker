package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Industry activities whose delivered items count towards lifetime totals.
const (
	ActivityManufacturing = 1
	ActivityReaction      = 11
	ActivityOldReaction   = 9 // reactions as ESI reported them before 2018
)

// DeliveredJob is one delivered manufacturing or reaction job and what it
// made.
type DeliveredJob struct {
	JobID           int64
	ActivityID      int
	InstallerID     int64
	InstallerName   string
	BlueprintTypeID int64
	ProductTypeID   int64
	ProductName     string
	Runs            int64
	UnitsPerRun     int64
	Quantity        int64
	FacilityID      int64
	CompletedDate   time.Time
}

// LoadUnrecordedDeliveries returns the delivered manufacturing and reaction
// jobs saved in industry_jobs that aren't in delivered_jobs yet, each job
// once. ProductName, InstallerName, UnitsPerRun and Quantity are left for
// the caller.
func LoadUnrecordedDeliveries(ctx context.Context) ([]DeliveredJob, error) {
	conn, err := connectDB()
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(ctx,
		`SELECT DISTINCT ON (j.job_id) j.job_id, j.activity_id, j.installer_id, j.blueprint_type_id,
			j.product_type_id, j.runs, j.facility_id, coalesce(j.completed_date, j.end_date)
		FROM lifetime_indy.industry_jobs j
		WHERE j.status = 'delivered' AND j.activity_id = ANY($1) AND j.product_type_id IS NOT NULL
			AND NOT EXISTS (SELECT 1 FROM lifetime_indy.delivered_jobs d WHERE d.job_id = j.job_id)
		ORDER BY j.job_id`,
		[]int{ActivityManufacturing, ActivityOldReaction, ActivityReaction})
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (DeliveredJob, error) {
		var job DeliveredJob
		err := row.Scan(&job.JobID, &job.ActivityID, &job.InstallerID, &job.BlueprintTypeID,
			&job.ProductTypeID, &job.Runs, &job.FacilityID, &job.CompletedDate)
		return job, err
	})
}

// SaveDeliveredJobs stores each job once; a job already stored is left as
// it is.
func SaveDeliveredJobs(ctx context.Context, jobs []DeliveredJob) error {
	batch := &pgx.Batch{}
	for _, job := range jobs {
		batch.Queue(`INSERT INTO lifetime_indy.delivered_jobs
			(job_id, activity_id, installer_id, installer_name, blueprint_type_id, product_type_id, product_name,
			runs, units_per_run, quantity, facility_id, completed_date)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (job_id) DO NOTHING`,
			job.JobID, job.ActivityID, job.InstallerID, job.InstallerName, job.BlueprintTypeID, job.ProductTypeID,
			job.ProductName, job.Runs, job.UnitsPerRun, job.Quantity, job.FacilityID, job.CompletedDate)
	}
	return sendBatch(ctx, batch)
}

// LoadUnpricedDeliveries returns the recorded jobs whose items have no
// price yet: JobID, ProductTypeID and CompletedDate only.
func LoadUnpricedDeliveries(ctx context.Context) ([]DeliveredJob, error) {
	conn, err := connectDB()
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(ctx,
		`SELECT job_id, product_type_id, completed_date FROM lifetime_indy.delivered_jobs WHERE unit_price IS NULL`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (DeliveredJob, error) {
		var job DeliveredJob
		err := row.Scan(&job.JobID, &job.ProductTypeID, &job.CompletedDate)
		return job, err
	})
}

// SetDeliveryPrices stores each job's item price, by job ID, and its value.
func SetDeliveryPrices(ctx context.Context, unitPrices map[int64]float64) error {
	batch := &pgx.Batch{}
	for jobID, price := range unitPrices {
		batch.Queue(`UPDATE lifetime_indy.delivered_jobs SET unit_price = $2, value = quantity * $2::numeric(20,2)
			WHERE job_id = $1`, jobID, price)
	}
	return sendBatch(ctx, batch)
}

// ProductTotal is how many of one item a user's characters have made and
// what they were worth when delivered.
type ProductTotal struct {
	ProductTypeID    int64
	ProductName      string
	Quantity         int64
	Value            float64 // leaves out UnpricedQuantity items
	UnpricedQuantity int64
	Jobs             int64
	FirstDelivered   time.Time
	LastDelivered    time.Time
}

// LoadLifetimeProduction adds up everything the characters started that was
// delivered, per item, most made first. activities picks manufacturing,
// reactions or both.
func LoadLifetimeProduction(ctx context.Context, installerIDs []int64, activities []int) ([]ProductTotal, error) {
	conn, err := connectDB()
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(ctx,
		`SELECT product_type_id, max(product_name), sum(quantity)::bigint, coalesce(sum(value), 0)::float8,
			coalesce(sum(quantity) FILTER (WHERE unit_price IS NULL), 0)::bigint, count(*),
			min(completed_date), max(completed_date)
		FROM lifetime_indy.delivered_jobs
		WHERE installer_id = ANY($1) AND activity_id = ANY($2)
		GROUP BY product_type_id
		ORDER BY sum(quantity) DESC, max(product_name)`,
		installerIDs, activities)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (ProductTotal, error) {
		var total ProductTotal
		err := row.Scan(&total.ProductTypeID, &total.ProductName, &total.Quantity, &total.Value,
			&total.UnpricedQuantity, &total.Jobs, &total.FirstDelivered, &total.LastDelivered)
		return total, err
	})
}
