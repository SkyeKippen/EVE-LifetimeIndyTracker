package esi

import (
	"context"
	"time"
)

// IndustryJob is a manufacturing, research, copying, invention or reaction
// job. Characters' jobs give their location as station_id and
// corporations' as location_id; both end up in LocationId.
type IndustryJob struct {
	JobId                int64     `json:"job_id"`
	ActivityId           int       `json:"activity_id"` // 1 manufacturing, 3 TE, 4 ME, 5 copying, 8 invention, 9 and 11 reactions
	Status               string    `json:"status"`      // active, cancelled, delivered, paused, ready or reverted
	InstallerId          int64     `json:"installer_id"`
	BlueprintId          int64     `json:"blueprint_id"`
	BlueprintTypeId      int64     `json:"blueprint_type_id"`
	BlueprintLocationId  int64     `json:"blueprint_location_id"`
	OutputLocationId     int64     `json:"output_location_id"`
	FacilityId           int64     `json:"facility_id"`
	LocationId           int64     `json:"location_id"`
	StationId            int64     `json:"station_id"`
	ProductTypeId        int64     `json:"product_type_id"`
	Runs                 int64     `json:"runs"`
	LicensedRuns         int64     `json:"licensed_runs"`
	SuccessfulRuns       int64     `json:"successful_runs"`
	Probability          float64   `json:"probability"`
	Cost                 float64   `json:"cost"`
	Duration             int64     `json:"duration"` // seconds
	StartDate            time.Time `json:"start_date"`
	EndDate              time.Time `json:"end_date"`
	PauseDate            time.Time `json:"pause_date"`
	CompletedDate        time.Time `json:"completed_date"`
	CompletedCharacterId int64     `json:"completed_character_id"`
}

// GetCharacterIndustryJobs returns the character's jobs, including ones
// finished in the last 90 days. Needs the
// esi-industry.read_character_jobs.v1 scope.
func (c *Client) GetCharacterIndustryJobs(ctx context.Context, characterId int64, accessToken string) ([]IndustryJob, error) {
	jobs, err := get[[]IndustryJob](ctx, c, c.url("/characters/%d/industry/jobs?include_completed=true", characterId), accessToken)
	for i := range jobs {
		if jobs[i].LocationId == 0 {
			jobs[i].LocationId = jobs[i].StationId
		}
	}
	return jobs, err
}

// GetCorporationIndustryJobs returns the corporation's jobs, including ones
// finished in the last 90 days. Needs the
// esi-industry.read_corporation_jobs.v1 scope and the Factory Manager role.
func (c *Client) GetCorporationIndustryJobs(ctx context.Context, corporationId int64, accessToken string) ([]IndustryJob, error) {
	return getPages[IndustryJob](ctx, c, c.url("/corporations/%d/industry/jobs?include_completed=true", corporationId), accessToken)
}
