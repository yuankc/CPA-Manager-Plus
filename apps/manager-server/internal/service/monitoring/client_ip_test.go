package monitoring

import (
	"context"
	"testing"

	"github.com/seakee/cpa-manager-plus/apps/manager-server/internal/store"
	"github.com/seakee/cpa-manager-plus/apps/manager-server/internal/usage"
)

func TestAnalyticsClientIPStatsPreferForwardedFor(t *testing.T) {
	db := newMonitoringTestStore(t)
	ctx := context.Background()
	fromMS := int64(1_778_000_000_000)
	toMS := fromMS + 60*60*1000
	latency := int64(200)

	if err := db.SaveModelPrices(ctx, map[string]store.ModelPrice{
		"gpt-a": {Prompt: 1, Completion: 2, Cache: 0.5},
	}); err != nil {
		t.Fatalf("save model prices: %v", err)
	}

	forwarded := monitoringEvent("ip-forwarded", fromMS+1_000, "gpt-a", "auth-1", "source-a", false, 100, 50, 0, 0, 150, &latency)
	forwarded.ClientIP = "10.0.0.1"
	forwarded.XForwardedFor = " 203.0.113.7 , 10.0.0.1"
	direct := monitoringEvent("ip-direct", fromMS+2_000, "gpt-b", "auth-1", "source-a", true, 10, 5, 0, 0, 15, nil)
	direct.ClientIP = "203.0.113.7"
	other := monitoringEvent("ip-other", fromMS+3_000, "gpt-a", "auth-2", "source-b", false, 1, 1, 0, 0, 2, nil)
	other.ClientIP = "198.51.100.2"
	unknown := monitoringEvent("ip-unknown", fromMS+4_000, "gpt-a", "auth-2", "source-b", false, 1, 1, 0, 0, 2, nil)

	if _, err := db.InsertEvents(ctx, []usage.Event{forwarded, direct, other, unknown}); err != nil {
		t.Fatalf("insert events: %v", err)
	}

	includeFailed := true
	resp, err := New(db).Analytics(ctx, Request{
		FromMS:  fromMS,
		ToMS:    toMS,
		NowMS:   toMS,
		Filters: Filters{IncludeFailed: &includeFailed},
		Include: Include{ClientIPStats: true, FilterOptions: true},
	})
	if err != nil {
		t.Fatalf("analytics: %v", err)
	}

	byIP := map[string]ClientIPStatRow{}
	for _, row := range resp.ClientIPStats {
		byIP[row.ClientIP] = row
	}
	if len(byIP) != 3 {
		t.Fatalf("client ip stats = %#v", resp.ClientIPStats)
	}
	primary := byIP["203.0.113.7"]
	if primary.Calls != 2 || primary.FailureCalls != 1 || primary.TotalTokens != 165 || len(primary.Models) != 2 {
		t.Fatalf("primary ip row = %#v", primary)
	}
	if primary.Cost <= 0 || primary.AvgLatencyMS == nil || *primary.AvgLatencyMS != 200 {
		t.Fatalf("primary ip cost/latency = %#v", primary)
	}
	if resp.ClientIPStats[0].ClientIP != "203.0.113.7" {
		t.Fatalf("expected busiest ip first, got %#v", resp.ClientIPStats[0])
	}
	if byIP[""].Calls != 1 {
		t.Fatalf("unknown ip row = %#v", byIP[""])
	}
	if resp.FilterOptions == nil || len(resp.FilterOptions.ClientIPs) != 2 {
		t.Fatalf("filter options client ips = %#v", resp.FilterOptions)
	}

	filtered, err := New(db).Analytics(ctx, Request{
		FromMS:  fromMS,
		ToMS:    toMS,
		NowMS:   toMS,
		Filters: Filters{IncludeFailed: &includeFailed, ClientIPs: []string{"203.0.113.7"}},
		Include: Include{Summary: true, ClientIPStats: true},
	})
	if err != nil {
		t.Fatalf("filtered analytics: %v", err)
	}
	if filtered.Summary == nil || filtered.Summary.TotalCalls != 2 || len(filtered.ClientIPStats) != 1 {
		t.Fatalf("filtered summary = %#v stats = %#v", filtered.Summary, filtered.ClientIPStats)
	}
}
