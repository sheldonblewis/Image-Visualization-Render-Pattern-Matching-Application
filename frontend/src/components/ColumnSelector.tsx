interface ColumnSelectorProps {
  value: number;
  onChange: (value: number) => void;
  disabled?: boolean;
}

const OPTIONS = [2, 4, 6, 8];

export function ColumnSelector({ value, onChange, disabled = false }: ColumnSelectorProps) {
  return (
    <div className="control">
      <label htmlFor="column-select">Columns</label>
      <select
        id="column-select"
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        disabled={disabled}
      >
        {OPTIONS.map((opt) => (
          <option key={opt} value={opt}>
            {opt}
          </option>
        ))}
      </select>
    </div>
  );
}
