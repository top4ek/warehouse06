import { CaretRightOutlined, InfoCircleOutlined, PauseOutlined } from "@ant-design/icons";
import { Button, Card, Flex, Popover, Typography } from "antd";
import { useMemo, useState } from "react";
import { Link as RouterLink, useNavigate } from "react-router-dom";
import { storageUrl, type DirectoryItem, type FileItem } from "../api";
import { useTapePlayer } from "../hooks/useTapePlayer";
import { formatBytes } from "../lib/format";
import { entryPlayLocation } from "../lib/playRoute";
import { canPlayWav } from "../lib/tapeAudio";

const PAGE_SIZE = 50;

type Props = {
  directories: DirectoryItem[];
  files: FileItem[];
  entryPath: string;
  isPlayable: (file: FileItem) => boolean;
};

type ListRow =
  | { kind: "parent"; path: string }
  | { kind: "dir"; name: string; path: string }
  | { kind: "file"; file: FileItem };

function parentPath(path: string): string | null {
  const i = path.lastIndexOf("/");
  if (i < 0) return null;
  return path.slice(0, i);
}

function PaginatedSection({
  title,
  total,
  page,
  totalPages,
  onPrev,
  onNext,
  children,
}: {
  title: string;
  total: number;
  page: number;
  totalPages: number;
  onPrev: () => void;
  onNext: () => void;
  children: React.ReactNode;
}) {
  return (
    <Card size="small" style={{ background: "var(--ant-color-fill-quaternary)" }}>
      <Flex justify="space-between" align="center">
        <Typography.Text type="secondary" style={{ textTransform: "uppercase", fontSize: 12 }}>
          {title}
        </Typography.Text>
        {total > PAGE_SIZE && (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {total}
          </Typography.Text>
        )}
      </Flex>
      <div style={{ marginTop: 12 }}>{children}</div>
      {totalPages > 1 && (
        <Flex justify="space-between" align="center" style={{ marginTop: 16 }}>
          <Button size="small" disabled={page === 0} onClick={onPrev}>
            Prev
          </Button>
          <Typography.Text type="secondary">
            {page + 1} / {totalPages}
          </Typography.Text>
          <Button size="small" disabled={page >= totalPages - 1} onClick={onNext}>
            Next
          </Button>
        </Flex>
      )}
    </Card>
  );
}

function FileInfoContent({ file }: { file: FileItem }) {
  const size = formatBytes(file.size);
  return (
    <Flex vertical gap={4} style={{ maxWidth: 320 }}>
      <Typography.Text>
        Size: {size ?? "-"}
      </Typography.Text>
      <div>
        <Typography.Text>SHA256:</Typography.Text>
        {file.sha256 ? (
          <Typography.Text
            code
            copyable
            style={{ display: "block", wordBreak: "break-all" }}
          >
            {file.sha256}
          </Typography.Text>
        ) : (
          <Typography.Text> -</Typography.Text>
        )}
      </div>
    </Flex>
  );
}

export default function EntrySidebarResources({
  directories,
  files,
  entryPath,
  isPlayable,
}: Props) {
  const navigate = useNavigate();
  const { playingKey, pendingKey, toggle } = useTapePlayer();
  const [page, setPage] = useState(0);

  const parent = parentPath(entryPath);
  const nonImageFiles = useMemo(() => files.filter((f) => !f.is_image), [files]);

  const rows = useMemo(() => {
    const items: ListRow[] = [];
    if (parent != null) {
      items.push({ kind: "parent", path: parent });
    }
    for (const dir of directories) {
      items.push({ kind: "dir", name: dir.name, path: dir.path });
    }
    for (const file of nonImageFiles) {
      items.push({ kind: "file", file });
    }
    return items;
  }, [parent, directories, nonImageFiles]);

  const totalPages = Math.max(1, Math.ceil(rows.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages - 1);
  const visibleRows = rows.slice(safePage * PAGE_SIZE, safePage * PAGE_SIZE + PAGE_SIZE);

  if (rows.length === 0) return null;

  return (
    <PaginatedSection
      title="Files"
      total={rows.length}
      page={safePage}
      totalPages={totalPages}
      onPrev={() => setPage((p) => p - 1)}
      onNext={() => setPage((p) => p + 1)}
    >
      <Flex vertical>
        {visibleRows.map((row, index) => {
          const rowStyle: React.CSSProperties = {
            padding: "8px 0",
            borderBlockEnd:
              index < visibleRows.length - 1 ? "1px solid var(--ant-color-split)" : undefined,
          };

          if (row.kind === "parent" || row.kind === "dir") {
            const label = row.kind === "parent" ? ".." : row.name;
            return (
              <div key={`${row.kind}-${row.path}`} style={rowStyle}>
                <RouterLink to={`/${row.path}`}>{label}</RouterLink>
              </div>
            );
          }

          const { file } = row;
          const rowKey = file.filepath || file.filename;
          const href = storageUrl(file.filepath || `${entryPath}/${file.filename}`);
          const playingTape = playingKey === rowKey;
          return (
            <Flex
              key={rowKey}
              justify="space-between"
              align="center"
              gap={8}
              style={rowStyle}
            >
              <a
                href={href}
                download={file.filename}
                style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap", minWidth: 0 }}
              >
                {file.filename}
              </a>
              <Flex gap={4} align="center" style={{ flexShrink: 0 }}>
                <Popover
                  trigger="click"
                  title="File info"
                  content={<FileInfoContent file={file} />}
                >
                  <Button size="small" icon={<InfoCircleOutlined />} aria-label="File info" />
                </Popover>
                {canPlayWav(file.filename) && (
                  <Button
                    size="small"
                    aria-label={playingTape ? "Stop tape audio" : "Play tape audio"}
                    loading={pendingKey === rowKey}
                    icon={playingTape ? <PauseOutlined /> : <CaretRightOutlined />}
                    onClick={() => toggle(rowKey, href, file.filename)}
                  />
                )}
                {isPlayable(file) && (
                  <Button
                    size="small"
                    type="primary"
                    onClick={() => navigate(entryPlayLocation(entryPath, file.filename))}
                  >
                    Run
                  </Button>
                )}
              </Flex>
            </Flex>
          );
        })}
      </Flex>
    </PaginatedSection>
  );
}
