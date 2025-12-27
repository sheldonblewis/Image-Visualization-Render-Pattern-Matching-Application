import { useEffect, useMemo, useState, useCallback, useRef } from 'react';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { PatternForm } from './components/PatternForm';
import { GridViewport } from './components/GridViewport';
import { ViewerModal } from './components/ViewerModal';
import { ColumnSelector } from './components/ColumnSelector';
import { groupMatches, MatchItem, GroupedResult } from './lib/transform';
import { QueryMode, QueryResponse, CountResponse } from './types/api';
import './App.css';

const DEFAULT_PATTERN = 'gs://wlt-public-sandbox/imgrid-takehome/%exp%/%class%_00.jpg';

export default function App() {
  const [pattern, setPattern] = useState(DEFAULT_PATTERN);
  const [mode, setMode] = useState<QueryMode>('percent');
  const [groupBy, setGroupBy] = useState<string | null>(null);
  const [columns, setColumns] = useState(4);
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
  const [hasSubmitted, setHasSubmitted] = useState(false);
  const [queryVersion, setQueryVersion] = useState(0);
  const [isViewerLoadingAhead, setIsViewerLoadingAhead] = useState(false);

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
    error,
  } = useInfiniteQuery<QueryResponse>({
    queryKey: ['matches', pattern, mode, queryVersion],
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
    enabled: hasSubmitted && !!pattern,
    refetchOnMount: false,
  });

  const {
    data: countData,
    isFetching: isCounting,
    error: countError,
  } = useQuery<CountResponse>({
    queryKey: ['count', pattern, mode, queryVersion],
    queryFn: async () => {
      const response = await fetch('/api/count', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pattern, mode, pageSize: 120 }),
      });
      if (!response.ok) {
        const body = await response.json().catch(() => ({}));
        throw new Error(body.error || 'Count failed');
      }
      return response.json();
    },
    enabled: hasSubmitted && !!pattern,
    staleTime: Infinity,
    refetchOnMount: false,
  });

  const captureNames = data?.pages[0]?.captureNames ?? [];
  const matches = useMemo(() => data?.pages.flatMap((page) => page.items) ?? [], [data]);
  const controlsDisabled = captureNames.length === 0 || isLoading;
  const allItemsLoaded = !hasNextPage && !isFetchingNextPage;
  const matchedTotal = countData?.total ?? null;

  useEffect(() => {
    if (captureNames.length > 0) {
      setGroupBy((prev) => (prev && captureNames.includes(prev) ? prev : captureNames[0]));
    } else {
      setGroupBy(null);
    }
  }, [captureNames]);

  const { rows, matches: groupedMatches } = useMemo<GroupedResult>(
    () => groupMatches(matches, groupBy && captureNames.includes(groupBy) ? groupBy : captureNames[0], columns),
    [matches, groupBy, captureNames, columns]
  );
  const totalFiles = groupedMatches.length;
  const captureCount = captureNames.length;
  const previousMatchCountRef = useRef(groupedMatches.length);

  const handleSelect = (item: MatchItem) => {
    setIsViewerLoadingAhead(false);
    setSelectedIndex(item.globalIndex);
  };

  const handleCloseViewer = () => {
    setSelectedIndex(null);
    setIsViewerLoadingAhead(false);
  };

  const triggerViewerPrefetch = useCallback(() => {
    if (!hasNextPage) {
      return;
    }
    setIsViewerLoadingAhead(true);
    if (!isFetchingNextPage) {
      fetchNextPage();
    }
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);

  const handleNavigate = useCallback(
    (direction: 1 | -1) => {
      setSelectedIndex((prev) => {
        if (groupedMatches.length === 0) {
          return prev;
        }

        const lastIndex = groupedMatches.length - 1;
        const sentinelIndex = groupedMatches.length;
        const currentIndex = prev ?? (direction === 1 ? 0 : lastIndex);

        if (direction === 1) {
          if (currentIndex === sentinelIndex) {
            return sentinelIndex;
          }
          if (currentIndex >= lastIndex) {
            if (!allItemsLoaded && hasNextPage) {
              triggerViewerPrefetch();
              return sentinelIndex;
            }
            setIsViewerLoadingAhead(false);
            return allItemsLoaded ? 0 : lastIndex;
          }
          setIsViewerLoadingAhead(false);
          return currentIndex + 1;
        }

        if (direction === -1) {
          if (currentIndex === sentinelIndex) {
            setIsViewerLoadingAhead(false);
            return lastIndex;
          }
          if (currentIndex <= 0) {
            setIsViewerLoadingAhead(false);
            return allItemsLoaded ? lastIndex : 0;
          }
          setIsViewerLoadingAhead(false);
          return currentIndex - 1;
        }

        return currentIndex;
      });
    },
    [groupedMatches.length, allItemsLoaded, hasNextPage, triggerViewerPrefetch]
  );

  useEffect(() => {
    const previousCount = previousMatchCountRef.current;
    if (isViewerLoadingAhead) {
      if (groupedMatches.length > previousCount) {
        setIsViewerLoadingAhead(false);
        setSelectedIndex(previousCount);
      } else if (!hasNextPage && !isFetchingNextPage) {
        setIsViewerLoadingAhead(false);
        setSelectedIndex((prev) => {
          if (prev === null) {
            return prev;
          }
          if (prev >= groupedMatches.length) {
            return groupedMatches.length > 0 ? groupedMatches.length - 1 : null;
          }
          return prev;
        });
      }
    } else if (selectedIndex !== null && selectedIndex >= groupedMatches.length && groupedMatches.length > 0) {
      setSelectedIndex(groupedMatches.length - 1);
    } else if (groupedMatches.length === 0 && selectedIndex !== null) {
      setSelectedIndex(null);
    }

    previousMatchCountRef.current = groupedMatches.length;
  }, [groupedMatches.length, hasNextPage, isFetchingNextPage, isViewerLoadingAhead, selectedIndex]);

  useEffect(() => {
    setSelectedIndex(null);
    setIsViewerLoadingAhead(false);
  }, [groupBy]);

  const handlePatternSubmit = useCallback(
    (nextPattern: string, nextMode: QueryMode) => {
      if (!nextPattern) {
        setHasSubmitted(false);
        setPattern('');
        setSelectedIndex(null);
        setIsViewerLoadingAhead(false);
        return;
      }

      setPattern(nextPattern);
      setMode(nextMode);
      setHasSubmitted(true);
      setQueryVersion((prevVersion) => prevVersion + 1);
      setSelectedIndex(null);
      setIsViewerLoadingAhead(false);
    },
    []
  );

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="title-block">
          <span className="eyebrow">Google Cloud Storage</span>
          <h1>Image Grid Viewer</h1>
        </div>
        <PatternForm value={pattern} mode={mode} onSubmit={handlePatternSubmit} />
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
              <ColumnSelector
                value={captureNames.length > 0 ? columns : null}
                onChange={setColumns}
                disabled={controlsDisabled}
              />
            </div>
            <div className="results-meta">
              <span>
                {captureCount} capture group{captureCount === 1 ? '' : 's'}
              </span>
              <span>
                {matchedTotal !== null
                  ? `${matchedTotal.toLocaleString()} files`
                  : isCounting
                  ? 'Counting files…'
                  : countError instanceof Error
                  ? 'Count unavailable'
                  : '—'}
              </span>
              <span className="muted">{totalFiles.toLocaleString()} loaded</span>
            </div>
          </div>

          <GridViewport
            rows={rows}
            isLoading={isLoading}
            error={error instanceof Error ? error.message : undefined}
            hasNextPage={hasNextPage}
            fetchNextPage={fetchNextPage}
            isFetchingNextPage={isFetchingNextPage}
            onSelect={handleSelect}
            columns={columns}
            captureNames={captureNames}
          />
        </section>
      )}

      <ViewerModal
        items={groupedMatches}
        selectedIndex={selectedIndex}
        captureNames={captureNames}
        allItemsLoaded={allItemsLoaded}
        isLoadingAhead={isViewerLoadingAhead}
        onClose={handleCloseViewer}
        onNavigate={handleNavigate}
      />
    </div>
  );
}
