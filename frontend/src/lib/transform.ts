import { QueryItem } from '../types/api';

export interface MatchItem extends QueryItem {
  groupKey: string;
  groupValue: string;
}

export type GridRow =
  | { type: 'header'; key: string; label: string }
  | { type: 'images'; key: string; groupKey: string; items: MatchItem[] };

export interface GroupedResult {
  rows: GridRow[];
  matches: MatchItem[];
}

export function groupMatches(items: QueryItem[], groupBy?: string, columns = 4): GroupedResult {
  if (!items.length || !groupBy) {
    return { rows: [], matches: [] };
  }

  const groups = new Map<string, MatchItem[]>();

  for (const item of items) {
    const value = item.captures[groupBy] ?? '—';
    const match: MatchItem = { ...item, groupKey: groupBy, groupValue: value };
    if (!groups.has(value)) {
      groups.set(value, []);
    }
    groups.get(value)!.push(match);
  }

  const sortedKeys = Array.from(groups.keys()).sort((a, b) => a.localeCompare(b));
  const rows: GridRow[] = [];
  const matches: MatchItem[] = [];

  for (const key of sortedKeys) {
    rows.push({ type: 'header', key: `header-${groupBy}-${key}`, label: `${groupBy}: ${key}` });
    const groupMatches = groups.get(key)!;
    matches.push(...groupMatches);
    for (let i = 0; i < groupMatches.length; i += columns) {
      rows.push({
        type: 'images',
        key: `row-${groupBy}-${key}-${i / columns}`,
        groupKey: groupBy,
        items: groupMatches.slice(i, i + columns),
      });
    }
  }

  return { rows, matches };
}
