import { useCallback, useEffect, useState } from 'react';
import { QueryMode } from '../types/api';

interface PatternFormProps {
  value: string;
  mode: QueryMode;
  onSubmit: (pattern: string, mode: QueryMode) => void;
}

export function PatternForm({ value, mode, onSubmit }: PatternFormProps) {
  const [draft, setDraft] = useState(value);
  const [currentMode, setCurrentMode] = useState<QueryMode>(mode);

  useEffect(() => {
    setDraft(value);
  }, [value]);

  useEffect(() => {
    setCurrentMode(mode);
  }, [mode]);

  const handleSubmit = useCallback(
    (event: React.FormEvent) => {
      event.preventDefault();
      onSubmit(draft.trim(), currentMode);
    },
    [draft, currentMode, onSubmit]
  );

  return (
    <form className="pattern-form" onSubmit={handleSubmit}>
      <div className="field stretch">
        <label htmlFor="pattern-input">Pattern</label>
        <textarea
          id="pattern-input"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          rows={1}
          spellCheck={false}
        />
      </div>
      <div className="field mode-picker">
        <label htmlFor="mode-select">Mode</label>
        <select id="mode-select" value={currentMode} onChange={(e) => setCurrentMode(e.target.value as QueryMode)}>
          <option value="percent">Percent</option>
          <option value="regex">Regex</option>
        </select>
      </div>
      <button type="submit">Run Query</button>
    </form>
  );
}
