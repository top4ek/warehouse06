import { useCallback, useState } from "react";

/**
 * useState persisted to localStorage. `validate` narrows the raw stored
 * string; returning null falls back to the (lazily computed) default.
 */
export function usePersistentState<T extends string>(
  key: string,
  getDefault: () => T,
  validate: (raw: string) => T | null,
) {
  const [value, setValueState] = useState<T>(() => {
    const raw = localStorage.getItem(key);
    if (raw !== null) {
      const parsed = validate(raw);
      if (parsed !== null) return parsed;
    }
    return getDefault();
  });

  const setValue = useCallback(
    (next: T) => {
      setValueState(next);
      localStorage.setItem(key, next);
    },
    [key],
  );

  return [value, setValue] as const;
}
