import { Link as RouterLink } from "react-router-dom";

type Item = { key: string; label: string; to: string };

export default function LinkList({ items }: { items: Item[] }) {
  return (
    <nav style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      {items.map(({ key, label, to }) => (
        <RouterLink key={key} to={to} style={{ fontWeight: 500 }}>
          {label}
        </RouterLink>
      ))}
    </nav>
  );
}
