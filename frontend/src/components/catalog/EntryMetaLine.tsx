import { Typography } from "antd";
import { formatAddedAt } from "../../lib/format";

type Props = {
  createdAt?: string;
  platform?: string;
  entryCount?: number;
  authorMode?: boolean;
};

export default function EntryMetaLine({ createdAt, platform, entryCount, authorMode }: Props) {
  const added = formatAddedAt(createdAt);
  const parts: string[] = [];

  if (authorMode && entryCount != null) {
    parts.push(`${entryCount} ${entryCount === 1 ? "work" : "works"}`);
  } else if (platform) {
    parts.push(platform);
  }
  if (added) parts.push(`Added ${added}`);

  if (parts.length === 0) return null;

  return (
    <Typography.Text type="secondary" ellipsis>
      {parts.join(" · ")}
    </Typography.Text>
  );
}
