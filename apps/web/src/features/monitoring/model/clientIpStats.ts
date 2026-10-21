import type { MonitoringAnalyticsClientIpStatRow } from '@/services/api/usageService';

export type ClientIpSortKey =
  | 'calls'
  | 'failure_calls'
  | 'success_rate'
  | 'total_tokens'
  | 'cost'
  | 'average_latency_ms'
  | 'last_seen_ms';

const sortValue = (row: MonitoringAnalyticsClientIpStatRow, key: ClientIpSortKey) => {
  const value = row[key];
  return typeof value === 'number' ? value : -1;
};

export const sortClientIpRows = (
  rows: MonitoringAnalyticsClientIpStatRow[],
  key: ClientIpSortKey,
  direction: 'asc' | 'desc'
) =>
  [...rows].sort((left, right) => {
    const delta = sortValue(left, key) - sortValue(right, key);
    if (delta !== 0) return direction === 'asc' ? delta : -delta;
    return right.calls - left.calls || left.client_ip.localeCompare(right.client_ip);
  });

export const filterClientIpRows = (rows: MonitoringAnalyticsClientIpStatRow[], keyword: string) => {
  const query = keyword.trim().toLowerCase();
  if (!query) return rows;
  return rows.filter((row) => row.client_ip.toLowerCase().includes(query));
};
