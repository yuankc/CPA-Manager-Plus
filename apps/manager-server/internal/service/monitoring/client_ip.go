package monitoring

import (
	"context"
	"sort"

	"github.com/seakee/cpa-manager-plus/apps/manager-server/internal/store"
)

type ClientIPStatRow struct {
	ID                  string                `json:"id"`
	ClientIP            string                `json:"client_ip"`
	Calls               int64                 `json:"calls"`
	SuccessCalls        int64                 `json:"success_calls"`
	FailureCalls        int64                 `json:"failure_calls"`
	SuccessRate         float64               `json:"success_rate"`
	InputTokens         int64                 `json:"input_tokens"`
	OutputTokens        int64                 `json:"output_tokens"`
	CachedTokens        int64                 `json:"cached_tokens"`
	CacheReadTokens     int64                 `json:"cache_read_tokens"`
	CacheCreationTokens int64                 `json:"cache_creation_tokens"`
	TotalTokens         int64                 `json:"total_tokens"`
	Cost                float64               `json:"cost"`
	AvgLatencyMS        *float64              `json:"average_latency_ms"`
	LastSeenMS          int64                 `json:"last_seen_ms"`
	APIKeyHashes        []string              `json:"api_key_hashes,omitempty"`
	Models              []AccountModelStatRow `json:"models,omitempty"`
}

type clientIPStatAccumulator struct {
	row            ClientIPStatRow
	apiKeyHashes   map[string]struct{}
	models         map[string]*AccountModelStatRow
	latencySum     float64
	latencySamples int64
}

func (s *Service) clientIPOptionValues(ctx context.Context, optionFilter store.AnalyticsFilter) ([]string, error) {
	return s.store.ClientIPFilterValuesWithFilter(ctx, optionFilter)
}

func buildClientIPStats(stats []store.ClientIPModelStat, prices map[string]store.ModelPrice) []ClientIPStatRow {
	grouped := map[string]*clientIPStatAccumulator{}
	for _, stat := range stats {
		entry := grouped[stat.ClientIP]
		if entry == nil {
			entry = &clientIPStatAccumulator{
				row:          ClientIPStatRow{ID: stat.ClientIP, ClientIP: stat.ClientIP},
				apiKeyHashes: map[string]struct{}{},
				models:       map[string]*AccountModelStatRow{},
			}
			grouped[stat.ClientIP] = entry
		}
		addSetValue(entry.apiKeyHashes, stat.APIKeyHash)
		cost := costForAPIKeyModelStat(stat.APIKeyModelStat, prices)
		addAccountTotals(
			&entry.row.Calls,
			&entry.row.SuccessCalls,
			&entry.row.FailureCalls,
			&entry.row.InputTokens,
			&entry.row.OutputTokens,
			&entry.row.CachedTokens,
			&entry.row.CacheReadTokens,
			&entry.row.CacheCreationTokens,
			&entry.row.TotalTokens,
			&entry.row.Cost,
			stat.Calls,
			stat.SuccessCalls,
			stat.FailureCalls,
			stat.InputTokens,
			stat.OutputTokens,
			stat.CachedTokens,
			stat.CacheReadTokens,
			stat.CacheCreationTokens,
			stat.TotalTokens,
			cost,
		)
		if stat.LastSeenMS > entry.row.LastSeenMS {
			entry.row.LastSeenMS = stat.LastSeenMS
		}
		if stat.AvgLatencyMS.Valid && stat.LatencySamples > 0 {
			entry.latencySum += stat.AvgLatencyMS.Float64 * float64(stat.LatencySamples)
			entry.latencySamples += stat.LatencySamples
		}
		addAccountModelStat(entry.models, stat.Model, stat.BillingModel, stat.Calls, stat.SuccessCalls, stat.FailureCalls, stat.InputTokens, stat.OutputTokens, stat.CachedTokens, stat.CacheReadTokens, stat.CacheCreationTokens, stat.TotalTokens, cost, stat.LastSeenMS)
	}

	result := make([]ClientIPStatRow, 0, len(grouped))
	for _, entry := range grouped {
		entry.row.SuccessRate = ratio(entry.row.SuccessCalls, entry.row.Calls)
		entry.row.APIKeyHashes = sortedSetValues(entry.apiKeyHashes)
		entry.row.Models = sortedAccountModelStats(entry.models)
		if entry.latencySamples > 0 {
			value := entry.latencySum / float64(entry.latencySamples)
			entry.row.AvgLatencyMS = &value
		}
		result = append(result, entry.row)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Calls != result[j].Calls {
			return result[i].Calls > result[j].Calls
		}
		if result[i].LastSeenMS != result[j].LastSeenMS {
			return result[i].LastSeenMS > result[j].LastSeenMS
		}
		return result[i].ClientIP < result[j].ClientIP
	})
	return result
}
