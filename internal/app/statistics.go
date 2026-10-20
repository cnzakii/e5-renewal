package app

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type DailyStatistics struct {
	Day       string `gorm:"column:local_day" json:"day"`
	Requests  int    `                        json:"requests"`
	Successes int    `                        json:"successes"`
}
type AccountStatistics struct {
	Requests  int               `json:"requests"`
	Successes int               `json:"successes"`
	Daily     []DailyStatistics `json:"daily"`
}

func requestLocation(w http.ResponseWriter, r *http.Request) (*time.Location, bool) {
	name := r.Header.Get("X-Time-Zone")
	if name == "" {
		return time.UTC, true
	}

	location, err := time.LoadLocation(name)
	if err != nil || name == "Local" {
		fail(w, 400, newError("time_zone_invalid", "Use an IANA time zone such as Asia/Shanghai."))
		return nil, false
	}

	return location, true
}

func statisticsStart(now time.Time, days int, location *time.Location) time.Time {
	now = now.In(location)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location).AddDate(0, 0, 1-days)
}

func (s *Server) statistics(ctx context.Context, accountID uint, start time.Time) (map[uint]AccountStatistics, error) {
	var rows []struct {
		AccountID uint
		DailyStatistics
	}
	// Local-midnight boundaries keep DST handling in Go and SQL identical across databases.
	var bucket strings.Builder
	bucket.WriteString("CASE ")
	args := make([]any, 0, 14)
	for i := 6; i >= 0; i-- {
		day := start.AddDate(0, 0, i)
		bucket.WriteString("WHEN graph_calls.started_at >= ? THEN ? ")
		args = append(args, day.UTC(), day.Format("2006-01-02"))
	}

	bucket.WriteString("END")
	query := s.db.WithContext(ctx).
		Model(&GraphCall{}).
		Joins("JOIN runs ON runs.id = graph_calls.run_id").
		Where("graph_calls.started_at >= ? AND graph_calls.started_at < ?", start.UTC(), start.AddDate(0, 0, 7).UTC())
	if accountID != 0 {
		query = query.Where("runs.account_id = ?", accountID)
	}

	const totals = "COUNT(*) AS requests, " +
		"SUM(CASE WHEN graph_calls.http_status >= 200 AND graph_calls.http_status < 300 " +
		"AND graph_calls.error = '' THEN 1 ELSE 0 END) AS successes"
	selection := "runs.account_id, " + bucket.String() + " AS local_day, " + totals
	err := query.Select(selection, args...).Group("runs.account_id, local_day").Order("local_day").Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	statistics := map[uint]AccountStatistics{}
	for _, row := range rows {
		value := statistics[row.AccountID]
		value.Requests += row.Requests
		value.Successes += row.Successes
		value.Daily = append(value.Daily, row.DailyStatistics)
		statistics[row.AccountID] = value
	}

	return statistics, nil
}

func fillStatistics(value AccountStatistics, start time.Time) AccountStatistics {
	days := make([]DailyStatistics, 0, 7)
	for day := start; len(days) < 7; day = day.AddDate(0, 0, 1) {
		entry := DailyStatistics{Day: day.Format("2006-01-02")}
		for _, stored := range value.Daily {
			if stored.Day == entry.Day {
				entry = stored
				break
			}
		}

		days = append(days, entry)
	}

	value.Daily = days

	return value
}
