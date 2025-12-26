import { useEffect, useMemo, useState } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { PatternForm } from './components/PatternForm';
import { GridViewport } from './components/GridViewport';
import { ViewerModal } from './components/ViewerModal';
import { ColumnSelector } from './components/ColumnSelector';
import { groupMatches, MatchItem, GroupedResult } from './lib/transform';
import { QueryMode, QueryResponse } from './types/api';
import './App.css';

const DEFAULT_PATTERN = 'gs://wlt-public-sandbox/imgrid-takehome/%exp%/%class%_00.jpg';

export default function App() {
  const [pattern, setPattern] = useState(DEFAULT_PATTERN);
  const [mode, setMode] = useState<QueryMode>('percent');
  const [groupBy, setGroupBy] = useState<string | null>(null);
  const [columns, setColumns] = useState(4);
  const [selected, setSelected] = useState<MatchItem | null>(null);

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    refetch,
    isLoading,
    error,
  } = useInfiniteQuery<QueryResponse>({
    queryKey: ['matches', pattern, mode],
    initialPageParam: null as string | null,
    queryFn: async ({ pageParam }) => {
      const response = await fetch('/api/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pattern, mode, pageSize: 120, cursor: pageParam }),
      });
      if (!response.ok) {
        const body = await response.json().catch(() => ({}));
        throw new Error(body.error || 'Query failed');
      }
      return response.json();
    },
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    enabled: !!pattern,
    refetchOnMount: false,
  });

  const captureNames = data?.pages[0]?.captureNames ?? [];
  const matches = useMemo(() => data?.pages.flatMap((page) => page.items) ?? [], [data]);
  const controlsDisabled = captureNames.length === 0 || isLoading;

  useEffect(() => {
    if (!groupBy && captureNames.length > 0) {
      setGroupBy(captureNames[0]);
    }
  }, [captureNames, groupBy]);

  const { rows, matches: groupedMatches, groupCount } = useMemo<GroupedResult>(
    () => groupMatches(matches, groupBy && captureNames.includes(groupBy) ? groupBy : captureNames[0], columns),
    [matches, groupBy, captureNames, columns]
  );
  const totalFiles = groupedMatches.length;

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="title-block">
          <span className="eyebrow">Image Grid Viewer</span>
          <h1>Pattern Matched Exploration</h1>
        </div>
        <PatternForm
          value={pattern}
          mode={mode}
          onSubmit={(nextPattern: string, nextMode: QueryMode) => {
            setPattern(nextPattern);
            setMode(nextMode);
            refetch({ throwOnError: false, cancelRefetch: true });
          }}
        />
      </header>

      {(captureNames.length > 0 || isLoading) && (
        <section className="results-shell">
          <div className="results-top">
            <div className="controls-bar">
              <div className={`control${controlsDisabled ? ' disabled' : ''}`}>
                <label htmlFor="group-by">Group by</label>
                <select
                  id="group-by"
                  value={groupBy ?? captureNames[0] ?? ''}
                  onChange={(e) => setGroupBy(e.target.value)}
                  disabled={controlsDisabled}
                >
                  {captureNames.map((name) => (
                    <option key={name} value={name}>
                      {name}
                    </option>
                  ))}
                </select>
              </div>
              <ColumnSelector value={columns} onChange={setColumns} disabled={controlsDisabled} />
            </div>
            <div className="results-meta">
              <span>{groupCount.toLocaleString()} groups</span>
              <span>{totalFiles.toLocaleString()} files</span>
            </div>
          </div>

          <GridViewport
            rows={rows}
            isLoading={isLoading}
            error={error instanceof Error ? error.message : undefined}
            hasNextPage={hasNextPage}
            fetchNextPage={fetchNextPage}
            isFetchingNextPage={isFetchingNextPage}
            onSelect={setSelected}
            columns={columns}
          />
        </section>
      )}

      <ViewerModal
        items={groupedMatches}
        selected={selected}
        onClose={() => setSelected(null)}
        onNavigate={(direction: 1 | -1) => {
          if (!selected || groupedMatches.length === 0) return;
          const idx = groupedMatches.findIndex((item: MatchItem) => item.object === selected.object);
          if (idx === -1) return;
          const nextIdx = (idx + direction + groupedMatches.length) % groupedMatches.length;
          setSelected(groupedMatches[nextIdx]);
        }}
      />
    </div>
  );
}
