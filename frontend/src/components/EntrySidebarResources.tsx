import { Button, Card, Flex, List, Typography } from "antd";
import { useMemo, useState } from "react";
import { Link as RouterLink, useNavigate } from "react-router-dom";
import { storageUrl, type DirectoryItem, type FileItem } from "../api";
import { entryPlayLocation } from "../lib/playRoute";

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

export default function EntrySidebarResources({
  directories,
  files,
  entryPath,
  isPlayable,
}: Props) {
  const navigate = useNavigate();
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
      <List
        size="small"
        dataSource={visibleRows}
        renderItem={(row) => {
          if (row.kind === "parent" || row.kind === "dir") {
            const label = row.kind === "parent" ? ".." : row.name;
            return (
              <List.Item>
                <RouterLink to={`/${row.path}`}>{label}</RouterLink>
              </List.Item>
            );
          }

          const { file } = row;
          const href = storageUrl(file.filepath || `${entryPath}/${file.filename}`);
          return (
            <List.Item
              actions={
                isPlayable(file)
                  ? [
                      <Button
                        key="play"
                        size="small"
                        type="primary"
                        onClick={() => navigate(entryPlayLocation(entryPath, file.filename))}
                      >
                        Play
                      </Button>,
                    ]
                  : undefined
              }
            >
              <a href={href} download={file.filename} style={{ overflow: "hidden", textOverflow: "ellipsis" }}>
                {file.filename}
              </a>
            </List.Item>
          );
        }}
      />
    </PaginatedSection>
  );
}
