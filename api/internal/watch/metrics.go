package watch

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/thoutmose/axon/api/internal/db"
)

// RegisterSourceAge adds axon_watch_oldest_overdue_seconds: how long past its
// next fetch the most overdue enabled source is, once that exceeds an hour,
// and 0 when none is. It reads the database at each scrape.
func RegisterSourceAge(reg prometheus.Registerer, q db.Querier) {
	reg.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "axon_watch_oldest_overdue_seconds",
		Help: "Age in seconds of the most overdue enabled watch source, counted from its next fetch time, when over an hour; 0 if none.",
	}, func() float64 { return oldestOverdue(q, time.Now()) }))
}

func oldestOverdue(q db.Querier, now time.Time) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var age float64
	err := q.QueryRow(ctx, `SELECT coalesce(extract(epoch FROM $1::timestamptz - min(s.next_fetch_at)), 0)::float8
		FROM watch_sources s JOIN users u ON u.id = s.user_id
		WHERE s.enabled AND NOT u.demo AND s.next_fetch_at < $1::timestamptz - interval '1 hour'`, now).Scan(&age)
	if err != nil {
		return 0
	}
	return age
}
