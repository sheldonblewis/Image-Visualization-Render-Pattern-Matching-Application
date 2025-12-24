interface ColumnSelectorProps {
  value: number;
  onChange: (value: number) => void;
}

const OPTIONS = [2, 4, 6, 8];

export function ColumnSelector({ value, onChange }: ColumnSelectorProps) {
  return (
    <div className="control">
      <label htmlFor="column-select">Columns</label>
      <select
        id="column-select"
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
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
