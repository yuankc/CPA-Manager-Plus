import { describe, expect, it } from 'vitest';
import type { MonitoringAnalyticsClientIpStatRow } from '@/services/api/usageService';
import { filterClientIpRows, sortClientIpRows } from './clientIpStats';
import { buildMonitoringCenterAnalyticsInclude } from './monitoringAnalyticsModel';
import { buildUsageAnalyticsInclude } from '@/features/usage-analytics/usageAnalyticsModel';

const row = (
  clientIp: string,
  overrides: Partial<MonitoringAnalyticsClientIpStatRow> = {}
): MonitoringAnalyticsClientIpStatRow => ({
  id: clientIp,
  client_ip: clientIp,
  calls: 1,
  success_calls: 1,
  failure_calls: 0,
  success_rate: 1,
  input_tokens: 0,
  output_tokens: 0,
  cached_tokens: 0,
  cache_read_tokens: 0,
  cache_creation_tokens: 0,
  total_tokens: 0,
  cost: 0,
  average_latency_ms: null,
  last_seen_ms: 0,
  ...overrides,
});

describe('client IP stats helpers', () => {
  it('sorts by the selected metric and keeps missing latency last in descending order', () => {
    const rows = [
      row('10.0.0.1', { calls: 5, average_latency_ms: 300 }),
      row('10.0.0.2', { calls: 9, average_latency_ms: null }),
      row('10.0.0.3', { calls: 2, average_latency_ms: 900 }),
    ];

    expect(sortClientIpRows(rows, 'calls', 'desc').map((item) => item.client_ip)).toEqual([
      '10.0.0.2',
      '10.0.0.1',
      '10.0.0.3',
    ]);
    expect(
      sortClientIpRows(rows, 'average_latency_ms', 'desc').map((item) => item.client_ip)
    ).toEqual(['10.0.0.3', '10.0.0.1', '10.0.0.2']);
    expect(sortClientIpRows(rows, 'calls', 'asc')[0].client_ip).toBe('10.0.0.3');
  });

  it('filters rows by IP substring', () => {
    const rows = [row('203.0.113.7'), row('198.51.100.2'), row('')];
    expect(filterClientIpRows(rows, ' 203.0 ').map((item) => item.client_ip)).toEqual([
      '203.0.113.7',
    ]);
    expect(filterClientIpRows(rows, '')).toHaveLength(3);
  });

  it('requests client IP stats only for the client IP tabs', () => {
    expect(buildMonitoringCenterAnalyticsInclude('clientIps', 'hour')).toMatchObject({
      client_ip_stats: true,
      api_key_stats: true,
    });
    expect(
      buildMonitoringCenterAnalyticsInclude('apiKeys', 'hour').client_ip_stats
    ).toBeUndefined();
    expect(buildUsageAnalyticsInclude('clientIps', 'hour')).toMatchObject({
      client_ip_stats: true,
    });
    expect(buildUsageAnalyticsInclude('apiKeys', 'hour').client_ip_stats).toBeUndefined();
  });
});
