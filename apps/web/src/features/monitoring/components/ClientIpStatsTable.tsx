import { Fragment, useMemo, useState, type ReactNode } from 'react';
import type { TFunction } from 'i18next';
import { IconChevronDown, IconChevronUp, IconSearch } from '@/components/ui/icons';
import type { MonitoringAnalyticsClientIpStatRow } from '@/services/api/usageService';
import { formatCompactNumber, formatUsd } from '@/utils/usage';
import { filterClientIpRows, sortClientIpRows, type ClientIpSortKey } from '../model/clientIpStats';
import styles from './ClientIpStatsTable.module.scss';

type ClientIpStatsTableProps = {
  rows: MonitoringAnalyticsClientIpStatRow[];
  hasPrices: boolean;
  locale: string;
  t: TFunction;
  emptyState?: ReactNode;
  resolveApiKeyLabel?: (apiKeyHash: string) => string;
  maxRows?: number;
};

const SORT_COLUMNS: ClientIpSortKey[] = [
  'calls',
  'failure_calls',
  'success_rate',
  'total_tokens',
  'cost',
  'average_latency_ms',
  'last_seen_ms',
];

const formatPercent = (value: number) => `${(value * 100).toFixed(1)}%`;

const formatLatency = (value: number | null | undefined) =>
  typeof value === 'number' && Number.isFinite(value) ? `${Math.round(value)} ms` : '-';

const formatDateTime = (value: number, locale: string) =>
  value > 0
    ? new Date(value).toLocaleString(locale, {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      })
    : '-';

const defaultApiKeyLabel = (hash: string) => (hash ? `sha256:${hash.slice(0, 12)}` : '');

export function ClientIpStatsTable({
  rows,
  hasPrices,
  locale,
  t,
  emptyState,
  resolveApiKeyLabel = defaultApiKeyLabel,
  maxRows,
}: ClientIpStatsTableProps) {
  const [keyword, setKeyword] = useState('');
  const [sortKey, setSortKey] = useState<ClientIpSortKey>('calls');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc');
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  const totalCalls = useMemo(() => rows.reduce((sum, row) => sum + row.calls, 0), [rows]);
  const knownCount = useMemo(() => rows.filter((row) => row.client_ip).length, [rows]);
  const visibleRows = useMemo(() => {
    const sorted = sortClientIpRows(filterClientIpRows(rows, keyword), sortKey, sortDirection);
    return maxRows ? sorted.slice(0, maxRows) : sorted;
  }, [keyword, maxRows, rows, sortDirection, sortKey]);

  const toggleSort = (key: ClientIpSortKey) => {
    if (key === sortKey) {
      setSortDirection((current) => (current === 'desc' ? 'asc' : 'desc'));
      return;
    }
    setSortKey(key);
    setSortDirection('desc');
  };

  const renderIp = (ip: string) =>
    ip ? (
      <span className={styles.ipText}>{ip}</span>
    ) : (
      <span className={`${styles.ipText} ${styles.unknown}`}>
        {t('client_ip_stats.unknown_ip')}
      </span>
    );

  if (rows.length === 0) {
    return (
      <div className={styles.wrapper}>
        {emptyState ?? <div className={styles.empty}>{t('client_ip_stats.empty')}</div>}
      </div>
    );
  }

  return (
    <div className={styles.wrapper}>
      <div className={styles.toolbar}>
        <span className={styles.meta}>
          {t('client_ip_stats.meta', { count: knownCount, calls: formatCompactNumber(totalCalls) })}
        </span>
        <label className={styles.search}>
          <IconSearch size={14} />
          <input
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
            placeholder={t('client_ip_stats.search_placeholder')}
            aria-label={t('client_ip_stats.search_placeholder')}
          />
        </label>
      </div>
      <div className={styles.tableScroll}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th>{t('client_ip_stats.column_ip')}</th>
              {SORT_COLUMNS.map((key) => (
                <th
                  key={key}
                  aria-sort={
                    sortKey === key
                      ? sortDirection === 'asc'
                        ? 'ascending'
                        : 'descending'
                      : 'none'
                  }
                >
                  <button
                    type="button"
                    className={`${styles.sortButton} ${sortKey === key ? styles.sortActive : ''}`}
                    onClick={() => toggleSort(key)}
                  >
                    {t(`client_ip_stats.column_${key}`)}
                    {sortKey === key ? (
                      sortDirection === 'desc' ? (
                        <IconChevronDown size={12} />
                      ) : (
                        <IconChevronUp size={12} />
                      )
                    ) : null}
                  </button>
                </th>
              ))}
              <th>{t('client_ip_stats.column_api_keys')}</th>
            </tr>
          </thead>
          <tbody>
            {visibleRows.map((row) => {
              const isExpanded = Boolean(expanded[row.id]);
              const apiKeys = row.api_key_hashes ?? [];
              const models = row.models ?? [];
              return (
                <Fragment key={row.id || '__unknown__'}>
                  <tr>
                    <td>
                      <div className={styles.ipCell}>
                        <button
                          type="button"
                          className={styles.expandButton}
                          aria-expanded={isExpanded}
                          aria-label={t(
                            isExpanded ? 'client_ip_stats.collapse' : 'client_ip_stats.expand'
                          )}
                          onClick={() =>
                            setExpanded((current) => ({ ...current, [row.id]: !isExpanded }))
                          }
                        >
                          {isExpanded ? <IconChevronUp size={12} /> : <IconChevronDown size={12} />}
                        </button>
                        {renderIp(row.client_ip)}
                      </div>
                    </td>
                    <td>
                      {formatCompactNumber(row.calls)}
                      <span className={styles.share}>
                        {formatPercent(totalCalls > 0 ? row.calls / totalCalls : 0)}
                      </span>
                    </td>
                    <td className={row.failure_calls > 0 ? styles.failure : undefined}>
                      {formatCompactNumber(row.failure_calls)}
                    </td>
                    <td className={row.success_rate >= 0.99 ? styles.success : undefined}>
                      {formatPercent(row.success_rate)}
                    </td>
                    <td>{formatCompactNumber(row.total_tokens)}</td>
                    <td>{hasPrices ? formatUsd(row.cost) : '-'}</td>
                    <td>{formatLatency(row.average_latency_ms)}</td>
                    <td>{formatDateTime(row.last_seen_ms, locale)}</td>
                    <td>{apiKeys.length}</td>
                  </tr>
                  {isExpanded ? (
                    <tr className={styles.detailRow}>
                      <td colSpan={SORT_COLUMNS.length + 2}>
                        <div className={styles.detailGrid}>
                          <div>
                            <h4>{t('client_ip_stats.models_title')}</h4>
                            <table className={styles.innerTable}>
                              <thead>
                                <tr>
                                  <th>{t('client_ip_stats.column_model')}</th>
                                  <th>{t('client_ip_stats.column_calls')}</th>
                                  <th>{t('client_ip_stats.column_failure_calls')}</th>
                                  <th>{t('client_ip_stats.column_total_tokens')}</th>
                                  <th>{t('client_ip_stats.column_cost')}</th>
                                </tr>
                              </thead>
                              <tbody>
                                {models.map((model) => (
                                  <tr key={model.model}>
                                    <td>{model.model || '-'}</td>
                                    <td>{formatCompactNumber(model.calls)}</td>
                                    <td>{formatCompactNumber(model.failure_calls)}</td>
                                    <td>{formatCompactNumber(model.total_tokens)}</td>
                                    <td>{hasPrices ? formatUsd(model.cost) : '-'}</td>
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                          </div>
                          <div>
                            <h4>{t('client_ip_stats.api_keys_title')}</h4>
                            {apiKeys.length > 0 ? (
                              <ul className={styles.keyList}>
                                {apiKeys.map((hash) => (
                                  <li key={hash}>
                                    {resolveApiKeyLabel(hash) || defaultApiKeyLabel(hash)}
                                  </li>
                                ))}
                              </ul>
                            ) : (
                              <span className={styles.meta}>
                                {t('client_ip_stats.no_api_keys')}
                              </span>
                            )}
                          </div>
                        </div>
                      </td>
                    </tr>
                  ) : null}
                </Fragment>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
