interface ColumnSelectorProps {
  value: number | null;
  onChange: (value: number) => void;
  disabled?: boolean;
}

const OPTIONS = [2, 4, 6, 8];

export function ColumnSelector({ value, onChange, disabled = false }: ColumnSelectorProps) {
  return (
    <div className={`control${disabled ? ' disabled' : ''}`}>
      <label htmlFor="column-select">Columns</label>
      <select
        id="column-select"
        value={value ?? ''}
        onChange={(e) => {
          const next = Number(e.target.value);
          if (!Number.isNaN(next)) {
            onChange(next);
          }
        }}
        disabled={disabled}
      >
        <option value="" disabled hidden />
        {OPTIONS.map((opt) => (
          <option key={opt} value={opt}>
            {opt}
          </option>
        ))}
      </select>
    </div>
  );
}
