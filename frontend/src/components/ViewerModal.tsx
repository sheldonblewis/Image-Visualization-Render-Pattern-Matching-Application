import { useEffect, useMemo } from 'react';
import { MatchItem } from '../lib/transform';

interface ViewerModalProps {
  items: MatchItem[];
  selected: MatchItem | null;
  onClose: () => void;
  onNavigate: (direction: 1 | -1) => void;
}

export function ViewerModal({ items, selected, onClose, onNavigate }: ViewerModalProps) {
  useEffect(() => {
    if (!selected) return;
    const handleKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      } else if (event.key === 'ArrowRight') {
        onNavigate(1);
      } else if (event.key === 'ArrowLeft') {
        onNavigate(-1);
      }
    };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [selected, onClose, onNavigate]);

  if (!selected) {
    return null;
  }

  const index = useMemo(() => items.findIndex((item) => item.object === selected.object), [items, selected]);

  return (
    <div className="viewer-overlay" onClick={onClose}>
      <div className="viewer-dialog" onClick={(e) => e.stopPropagation()}>
        <header>
          <div>
            <p className="eyebrow">{selected.groupKey}</p>
            <h2>{selected.groupValue}</h2>
            <p className="meta">{selected.object}</p>
          </div>
          <button className="ghost" onClick={onClose} aria-label="Close">
            ×
          </button>
        </header>
        <div className="viewer-body">
          <button className="nav prev" onClick={() => onNavigate(-1)} aria-label="Previous image">
            ‹
          </button>
          <img src={selected.url} alt={selected.object} />
          <button className="nav next" onClick={() => onNavigate(1)} aria-label="Next image">
            ›
          </button>
        </div>
        <footer>
          <span>
            {index + 1} / {items.length}
          </span>
        </footer>
      </div>
    </div>
  );
}
