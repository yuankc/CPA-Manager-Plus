package usageevent

import (
	"context"
	"fmt"
)

// SQLClientIPExpression resolves the downstream client address for an event:
// the first X-Forwarded-For hop when present, otherwise the recorded client IP.
func SQLClientIPExpression(prefix string) string {
	xff := fmt.Sprintf("coalesce(%sx_forwarded_for, '')", prefix)
	return fmt.Sprintf(
		"coalesce(nullif(trim(case when instr(%[1]s, ',') > 0 then substr(%[1]s, 1, instr(%[1]s, ',') - 1) else %[1]s end), ''), nullif(trim(coalesce(%[2]sclient_ip, '')), ''), '')",
		xff,
		prefix,
	)
}

var clientIPExpr = SQLClientIPExpression("")

// ClientIPModelStat aggregates usage per resolved client IP, API key and
// pricing band so callers can price and group the rows by client address.
type ClientIPModelStat struct {
	APIKeyModelStat
	ClientIP string
}

func (r *repository) ClientIPModelStatsWithFilter(ctx context.Context, filter AnalyticsFilter) ([]ClientIPModelStat, error) {
	where, args := analyticsWhere(filter)
	rows, err := r.db.QueryContext(ctx, pricingBandedUsageEventsCTE+`
select
	`+clientIPExpr+` as client_ip_value,
	coalesce(api_key_hash, ''),
	analytics_model_value as model,
	billing_model_value as billing_model,
	pricing_model_value,
	context_threshold_tokens_value,
	coalesce(service_tier, '') as service_tier,
	count(*),
	sum(case when failed = 0 then 1 else 0 end),
	sum(case when failed = 1 then 1 else 0 end),
	coalesce(sum(`+normalizedInputExpr+`), 0),
	coalesce(sum(output_tokens), 0),
	coalesce(sum(`+compatCachedExpr+`), 0),
	coalesce(sum(cache_read_tokens), 0),
	coalesce(sum(cache_creation_tokens), 0),
	coalesce(sum(`+longInputExpr+`), 0),
	coalesce(sum(`+longOutputExpr+`), 0),
	coalesce(sum(`+longCachedExpr+`), 0),
	coalesce(sum(`+longCacheReadExpr+`), 0),
	coalesce(sum(`+longCacheCreationExpr+`), 0),
	coalesce(sum(total_tokens), 0),
	max(timestamp_ms),
	avg(nullif(latency_ms, 0)),
	count(nullif(latency_ms, 0))
from banded_usage_events `+where+`
group by client_ip_value, coalesce(api_key_hash, ''), analytics_model_value, billing_model, pricing_model_value, context_threshold_tokens_value, coalesce(service_tier, '')
order by max(timestamp_ms) desc, count(*) desc`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]ClientIPModelStat, 0)
	for rows.Next() {
		var stat ClientIPModelStat
		if err := rows.Scan(
			&stat.ClientIP,
			&stat.APIKeyHash,
			&stat.Model,
			&stat.BillingModel,
			&stat.PricingModel,
			&stat.ContextThresholdTokens,
			&stat.ServiceTier,
			&stat.Calls,
			&stat.SuccessCalls,
			&stat.FailureCalls,
			&stat.InputTokens,
			&stat.OutputTokens,
			&stat.CachedTokens,
			&stat.CacheReadTokens,
			&stat.CacheCreationTokens,
			&stat.LongInputTokens,
			&stat.LongOutputTokens,
			&stat.LongCachedTokens,
			&stat.LongCacheReadTokens,
			&stat.LongCacheCreationTokens,
			&stat.TotalTokens,
			&stat.LastSeenMS,
			&stat.AvgLatencyMS,
			&stat.LatencySamples,
		); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

func (r *repository) ClientIPFilterValuesWithFilter(ctx context.Context, filter AnalyticsFilter) ([]string, error) {
	return r.distinctFilterValues(ctx, filter, clientIPExpr)
}
