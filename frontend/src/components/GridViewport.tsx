import { memo, useCallback, useEffect, useRef, useState } from 'react';
import { VariableSizeList, ListOnItemsRenderedProps } from 'react-window';
import { GridRow, MatchItem } from '../lib/transform';
import { useWindowSize } from '../hooks/useWindowSize';

const HEADER_HEIGHT = 48;
const IMAGE_ROW_HEIGHT = 240;

interface GridViewportProps {
  rows: GridRow[];
  isLoading: boolean;
  error?: string;
  hasNextPage?: boolean;
  fetchNextPage: () => Promise<unknown> | void;
  isFetchingNextPage: boolean;
  onSelect: (item: MatchItem) => void;
  columns: number;
  captureNames: string[];
}

interface RowProps {
  row: GridRow;
  style: React.CSSProperties;
  onSelect: (item: MatchItem) => void;
}

const ThumbCard = memo(
  ({ item, onSelect, displayLabel }: { item: MatchItem; onSelect: (item: MatchItem) => void; displayLabel: string }) => {
    const [status, setStatus] = useState<'loading' | 'loaded' | 'error'>('loading');

    return (
      <button className={`image-card ${status}`} onClick={() => status !== 'error' && onSelect(item)}>
        <div className="thumb">
          {status === 'error' ? (
            <span className="thumb-error">Failed to load</span>
          ) : (
            <img
              src={item.url}
              alt={item.object}
              loading="lazy"
              onLoad={() => setStatus('loaded')}
              onError={() => setStatus('error')}
            />
          )}
        </div>
        <div className="meta">
          <span className="meta-primary">{displayLabel}</span>
          <span className="meta-secondary" title={item.object}>
            {item.object}
          </span>
        </div>
      </button>
    );
  }
);

const ListRow = memo(
  ({ row, style, onSelect, columns, captureNames }: RowProps & { columns: number; captureNames: string[] }) => {
    if (!row) return null;

    if (row.type === 'header') {
      return (
        <div className="grid-row header-row" style={style}>
          <span>{row.label}</span>
        </div>
      );
    }

    const placeholders = Math.max(0, columns - row.items.length);
    return (
      <div
        className="grid-row image-row"
        style={{
          ...style,
          display: 'grid',
          gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
          gap: '1rem',
        }}
      >
        {row.items.map((item) => {
          const secondaryKey = captureNames.find((name) => name !== item.groupKey) ?? item.groupKey;
          const secondaryValue = item.captures[secondaryKey] ?? item.groupValue;
          const label = secondaryKey ? `${secondaryKey}: ${secondaryValue}` : secondaryValue;
          return <ThumbCard key={item.object} item={item} onSelect={onSelect} displayLabel={label} />;
        })}
        {Array.from({ length: placeholders }).map((_, idx) => (
          <div key={`placeholder-${row.key}-${idx}`} className="image-card placeholder" aria-hidden="true" />
        ))}
      </div>
    );
  }
);

export function GridViewport({
  rows,
  isLoading,
  error,
  hasNextPage,
  fetchNextPage,
  isFetchingNextPage,
  onSelect,
  columns,
  captureNames,
}: GridViewportProps) {
  const { height } = useWindowSize();
  const listHeight = Math.max(360, height - 320);
  const listRef = useRef<VariableSizeList>(null);

  useEffect(() => {
    listRef.current?.resetAfterIndex(0, true);
  }, [rows]);

  const getItemSize = useCallback((index: number) => {
    const row = rows[index];
    return row?.type === 'header' ? HEADER_HEIGHT : IMAGE_ROW_HEIGHT;
  }, [rows]);

  const itemKey = useCallback((index: number) => rows[index]?.key ?? `row-${index}`, [rows]);

  const handleItemsRendered = useCallback(
    ({ visibleStopIndex }: ListOnItemsRenderedProps) => {
      if (!hasNextPage || isFetchingNextPage) return;
      if (visibleStopIndex >= rows.length - 8) {
        fetchNextPage();
      }
    },
    [fetchNextPage, hasNextPage, isFetchingNextPage, rows.length]
  );

  if (error) {
    return <div className="panel error">{error}</div>;
  }

  if (!rows.length && isLoading) {
    return <div className="panel status">Scanning bucket…</div>;
  }

  if (!rows.length) {
    return <div className="panel status">No matches yet. Try adjusting your pattern.</div>;
  }

  return (
    <div className="grid-viewport">
      <VariableSizeList
        ref={listRef}
        height={listHeight}
        width="100%"
        itemCount={rows.length}
        itemSize={getItemSize}
        itemKey={itemKey}
        overscanCount={6}
        onItemsRendered={handleItemsRendered}
      >
        {({ index, style }) => (
          <ListRow row={rows[index]} style={style} onSelect={onSelect} columns={columns} captureNames={captureNames} />
        )}
      </VariableSizeList>
      {isFetchingNextPage && (
        <div className="panel status subtle loading-inline">
          <span>Loading more…</span>
          <div className="progress">
            <div className="progress-bar" />
          </div>
        </div>
      )}
    </div>
  );
}
