import { useEffect, useMemo } from 'react';
import { MatchItem } from '../lib/transform';

interface ViewerModalProps {
  items: MatchItem[];
  selectedIndex: number | null;
  captureNames: string[];
  allItemsLoaded: boolean;
  isLoadingAhead: boolean;
  onClose: () => void;
  onNavigate: (direction: 1 | -1) => void;
}

export function ViewerModal({
  items,
  selectedIndex,
  captureNames,
  allItemsLoaded,
  isLoadingAhead,
  onClose,
  onNavigate,
}: ViewerModalProps) {
  const total = items.length;
  const showLoadingCard = Boolean(isLoadingAhead && selectedIndex === total);

  const selected = useMemo(() => {
    if (selectedIndex === null || showLoadingCard) return null;
    return items[selectedIndex] ?? null;
  }, [items, selectedIndex, showLoadingCard]);

  const captureEntries = useMemo(() => {
    if (!selected) return [];
    const ordered = captureNames.length ? captureNames : Object.keys(selected.captures ?? {});
    return ordered.map((name) => ({
      name,
      value: selected.captures?.[name] ?? '—',
    }));
  }, [selected, captureNames]);

  const wrapEnabled = allItemsLoaded && total > 0;
  const disablePrev = !wrapEnabled && selectedIndex === 0;
  const disableNext = showLoadingCard || (!wrapEnabled && selectedIndex !== null && selectedIndex >= total - 1);

  useEffect(() => {
    if (selectedIndex === null) return;
    const handleKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      } else if (event.key === 'ArrowRight' && !disableNext) {
        onNavigate(1);
      } else if (event.key === 'ArrowLeft' && !(disablePrev && !showLoadingCard)) {
        onNavigate(-1);
      }
    };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [selectedIndex, onClose, onNavigate, disableNext, disablePrev, showLoadingCard]);

  if (selectedIndex === null) {
    return null;
  }

  if (!showLoadingCard && !selected) {
    return null;
  }

  return (
    <div className="viewer-overlay" onClick={onClose}>
      <div className="viewer-dialog" onClick={(e) => e.stopPropagation()}>
        <header>
          <div className="viewer-header-meta">
            {showLoadingCard ? (
              <>
                <div className="loading-pill subtle">Loading more…</div>
                <p className="meta path">Fetching the next images…</p>
              </>
            ) : (
              <>
                <div className="capture-list" aria-label="Capture groups">
                  {captureEntries.map(({ name, value }) => (
                    <span key={name} className="capture-pill">
                      <strong>{name}:</strong> {value}
                    </span>
                  ))}
                </div>
                <p className="meta path">{selected?.object}</p>
              </>
            )}
          </div>
          <button className="ghost" onClick={onClose} aria-label="Close">
            ×
          </button>
        </header>
        <div className={`viewer-body${showLoadingCard ? ' loading' : ''}`}>
          <button
            className="nav prev"
            onClick={() => onNavigate(-1)}
            aria-label="Previous image"
            disabled={disablePrev && !showLoadingCard}
          >
            ‹
          </button>
          {showLoadingCard ? (
            <div className="viewer-loading-card" aria-live="polite">
              <div className="loading-pill">Loading more…</div>
              <p>Fetching the next batch of matches.</p>
            </div>
          ) : (
            <img src={selected?.url} alt={selected?.object ?? 'Selected match'} />
          )}
          <button
            className="nav next"
            onClick={() => onNavigate(1)}
            aria-label="Next image"
            disabled={disableNext}
          >
            ›
          </button>
        </div>
        <footer>
          <div className="viewer-footer-meta">
            <div className="key-hint">
              <span className="key">←</span>
              <span className="key">→</span>
              <span>{wrapEnabled ? 'Wrap enabled' : 'Wrap locked'}</span>
            </div>
            <span>{showLoadingCard ? `${items.length} loaded` : `${selectedIndex + 1} / ${items.length}`}</span>
          </div>
        </footer>
      </div>
    </div>
  );
}
