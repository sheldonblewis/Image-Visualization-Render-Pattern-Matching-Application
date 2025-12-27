import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
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

type LoadingRow = { type: 'loading'; key: 'loading' };
type ViewportRow = GridRow | LoadingRow;

interface RowProps {
  row: ViewportRow;
  style: React.CSSProperties;
  onSelect: (item: MatchItem) => void;
}

const ThumbCard = memo(
  ({ item, onSelect, displayLabel }: { item: MatchItem; onSelect: (item: MatchItem) => void; displayLabel: string }) => {
    const [status, setStatus] = useState<'loading' | 'loaded' | 'error'>('loading');

    return (
      <button
        type="button"
        className={`image-card ${status}`}
        onClick={() => status !== 'error' && onSelect(item)}
      >
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

    if (row.type === 'loading') {
      return (
        <div
          className="grid-row image-row loading-row"
          style={{
            ...style,
            display: 'grid',
            gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
            gap: '1rem',
          }}
          aria-hidden="true"
        >
          {Array.from({ length: columns }).map((_, idx) => (
            <div key={`loading-tile-${idx}`} className="image-card placeholder shimmer-tile">
              <div className="thumb shimmer-block" />
              <div className="meta">
                <span className="skeleton-line short" />
                <span className="skeleton-line" />
              </div>
            </div>
          ))}
        </div>
      );
    }

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
  const viewportRows = useMemo<ViewportRow[]>(() => {
    if (hasNextPage || isFetchingNextPage) {
      return [...rows, { type: 'loading', key: 'loading' }];
    }
    return rows;
  }, [rows, hasNextPage, isFetchingNextPage]);

  useEffect(() => {
    listRef.current?.resetAfterIndex(0, true);
  }, [viewportRows]);

  const getItemSize = useCallback(
    (index: number) => {
      const row = viewportRows[index];
      return row?.type === 'header' ? HEADER_HEIGHT : IMAGE_ROW_HEIGHT;
    },
    [viewportRows]
  );

  const itemKey = useCallback((index: number) => viewportRows[index]?.key ?? `row-${index}`, [viewportRows]);

  const handleItemsRendered = useCallback(
    ({ visibleStopIndex }: ListOnItemsRenderedProps) => {
      if (!hasNextPage || isFetchingNextPage) return;
      if (visibleStopIndex >= viewportRows.length - 8) {
        fetchNextPage();
      }
    },
    [fetchNextPage, hasNextPage, isFetchingNextPage, viewportRows.length]
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
        itemCount={viewportRows.length}
        itemSize={getItemSize}
        itemKey={itemKey}
        overscanCount={6}
        onItemsRendered={handleItemsRendered}
      >
        {({ index, style }) => (
          <ListRow
            row={viewportRows[index]}
            style={style}
            onSelect={onSelect}
            columns={columns}
            captureNames={captureNames}
          />
        )}
      </VariableSizeList>
    </div>
  );
}
