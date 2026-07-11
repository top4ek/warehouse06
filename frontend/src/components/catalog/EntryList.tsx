import { Avatar, Card, Col, Flex, Row, Tag, Typography } from "antd";
import { Link as RouterLink, useNavigate } from "react-router-dom";
import { storageUrl, type Entry } from "../../api";
import { browsePath } from "../../lib/browse";
import type { ViewMode } from "../../context/PrefsContext";
import { sanitizeHtml } from "../../lib/sanitizeHtml";
import EntryMetaLine from "./EntryMetaLine";

type Props = {
  entries: Entry[];
  viewMode: ViewMode;
  authorLinks?: boolean;
  onTagSelect?: (tag: string) => void;
  onEntryClick?: (entryPath: string, indexInWindow: number) => void;
};

function entryHref(entry: Entry, authorLinks: boolean) {
  return authorLinks
    ? `/authors/${entry.path.replace(/^authors\//, "")}`
    : `/${entry.path}`;
}

function TagChip({ tag, onTagSelect }: { tag: string; onTagSelect?: (tag: string) => void }) {
  const navigate = useNavigate();
  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (onTagSelect) onTagSelect(tag);
    else navigate(browsePath({ tag }));
  };
  return (
    <Tag style={{ cursor: "pointer" }} onClick={handleClick}>
      {tag}
    </Tag>
  );
}

function EntryDescription({
  entry,
  ellipsisRows,
  style,
}: {
  entry: Entry;
  ellipsisRows?: number;
  style?: React.CSSProperties;
}) {
  const html = entry.description_html || entry.description;
  if (!html) return null;

  const clampStyle: React.CSSProperties | undefined =
    ellipsisRows && ellipsisRows > 0
      ? {
          display: "-webkit-box",
          WebkitLineClamp: ellipsisRows,
          WebkitBoxOrient: "vertical",
          overflow: "hidden",
        }
      : undefined;

  return (
    <div
      className="content-html"
      style={{
        color: "var(--ant-color-text-secondary)",
        marginTop: 8,
        ...clampStyle,
        ...style,
      }}
      dangerouslySetInnerHTML={{ __html: sanitizeHtml(html) }}
    />
  );
}

function EntryPreview({ entry }: { entry: Entry }) {
  if (entry.preview_image) {
    return (
      <Avatar
        shape="square"
        src={storageUrl(entry.preview_image)}
        alt=""
        size={64}
        style={{ width: 64, height: 48, flexShrink: 0 }}
      />
    );
  }
  return (
    <Avatar shape="square" size={64} style={{ width: 64, height: 48, flexShrink: 0 }}>
      {(entry.name || entry.path || "?").slice(0, 1)}
    </Avatar>
  );
}

type EntryLinkProps = {
  entry: Entry;
  index: number;
  authorLinks?: boolean;
  onEntryClick?: (entryPath: string, indexInWindow: number) => void;
  style?: React.CSSProperties;
  className?: string;
  children: React.ReactNode;
};

function EntryLink({ entry, index, authorLinks = false, onEntryClick, style, className, children }: EntryLinkProps) {
  const href = entryHref(entry, authorLinks);

  if (authorLinks || !onEntryClick) {
    return (
      <RouterLink to={href} style={style} className={className} data-entry-path={entry.path}>
        {children}
      </RouterLink>
    );
  }

  return (
    <RouterLink
      to={href}
      style={style}
      className={className}
      data-entry-path={entry.path}
      onClick={(e) => {
        e.preventDefault();
        onEntryClick(entry.path, index);
      }}
    >
      {children}
    </RouterLink>
  );
}

function EntryListCompact({
  entries,
  authorLinks,
  onTagSelect,
  onEntryClick,
}: Pick<Props, "entries" | "authorLinks" | "onTagSelect" | "onEntryClick">) {
  return (
    <Card>
      {entries.map((entry, index) => (
        <Flex
          key={entry.path}
          gap={16}
          align="flex-start"
          data-entry-path={entry.path}
          style={{
            padding: "12px 0",
            borderBlockEnd:
              index < entries.length - 1 ? "1px solid var(--ant-color-split)" : undefined,
          }}
        >
          <EntryPreview entry={entry} />
          <Flex vertical style={{ flex: 1, minWidth: 0 }}>
            <EntryLink entry={entry} index={index} authorLinks={authorLinks} onEntryClick={onEntryClick} style={{ color: "inherit" }}>
              {entry.name || entry.path}
            </EntryLink>
            <EntryMetaLine
              createdAt={entry.created_at}
              platform={entry.platform || entry.path}
              entryCount={entry.entry_count}
              authorMode={authorLinks}
            />
          </Flex>
          {entry.tags && entry.tags.length > 0 && (
            <Flex gap={4} wrap="wrap" justify="flex-end" style={{ maxWidth: "40%" }}>
              {entry.tags.map((t) => (
                <TagChip key={t.name} tag={t.name} onTagSelect={onTagSelect} />
              ))}
            </Flex>
          )}
        </Flex>
      ))}
    </Card>
  );
}

function EntryListComfortable({
  entries,
  authorLinks,
  onTagSelect,
  onEntryClick,
}: Pick<Props, "entries" | "authorLinks" | "onTagSelect" | "onEntryClick">) {
  return (
    <Flex vertical gap={16}>
      {entries.map((entry, index) => (
        <Card key={entry.path} hoverable data-entry-path={entry.path}>
          <EntryLink
            entry={entry}
            index={index}
            authorLinks={authorLinks}
            onEntryClick={onEntryClick}
            style={{ color: "inherit", textDecoration: "none", display: "block" }}
          >
            <Flex gap={16}>
              {entry.preview_image && (
                <img
                  src={storageUrl(entry.preview_image)}
                  alt=""
                  width={128}
                  height={96}
                  loading="lazy"
                  decoding="async"
                  style={{ width: 128, height: 96, borderRadius: 8, objectFit: "cover", flexShrink: 0 }}
                />
              )}
              <Flex vertical style={{ flex: 1, minWidth: 0 }}>
                <Typography.Text strong style={{ fontSize: 16 }}>
                  {entry.name || entry.path}
                </Typography.Text>
                <EntryMetaLine
                  createdAt={entry.created_at}
                  platform={entry.platform}
                  entryCount={entry.entry_count}
                  authorMode={authorLinks}
                />
                <EntryDescription entry={entry} ellipsisRows={1} />
                {entry.tags && entry.tags.length > 0 && (
                  <Flex gap={4} wrap="wrap" style={{ marginTop: 12 }}>
                    {entry.tags.map((t) => (
                      <TagChip key={t.name} tag={t.name} onTagSelect={onTagSelect} />
                    ))}
                  </Flex>
                )}
              </Flex>
            </Flex>
          </EntryLink>
        </Card>
      ))}
    </Flex>
  );
}

function EntryListGrid({
  entries,
  authorLinks,
  onTagSelect,
  onEntryClick,
}: Pick<Props, "entries" | "authorLinks" | "onTagSelect" | "onEntryClick">) {
  return (
    <Row gutter={[16, 16]}>
      {entries.map((entry, index) => (
        <Col key={entry.path} xs={24} sm={12} lg={8}>
          <EntryLink
            entry={entry}
            index={index}
            authorLinks={authorLinks}
            onEntryClick={onEntryClick}
            style={{ display: "block", color: "inherit", textDecoration: "none" }}
          >
            <Card
              hoverable
              data-entry-path={entry.path}
              cover={
                <div
                  style={{
                    aspectRatio: "4/3",
                    background: "var(--ant-color-fill-quaternary)",
                    overflow: "hidden",
                  }}
                >
                  {entry.preview_image ? (
                    <img
                      src={storageUrl(entry.preview_image)}
                      alt=""
                      loading="lazy"
                      decoding="async"
                      style={{ width: "100%", height: "100%", objectFit: "cover" }}
                    />
                  ) : (
                    <Flex align="center" justify="center" style={{ height: "100%" }}>
                      <Typography.Title level={2} type="secondary" style={{ margin: 0 }}>
                        {(entry.name || entry.path || "?").slice(0, 1)}
                      </Typography.Title>
                    </Flex>
                  )}
                </div>
              }
              style={{ height: "100%" }}
            >
              <Flex vertical gap={6}>
                <Typography.Text strong style={{ fontSize: 16 }}>
                  {entry.name || entry.path}
                </Typography.Text>
                <EntryMetaLine
                  createdAt={entry.created_at}
                  platform={entry.platform}
                  entryCount={entry.entry_count}
                  authorMode={authorLinks}
                />
              </Flex>
              <EntryDescription entry={entry} ellipsisRows={3} style={{ marginBottom: 0 }} />
              {entry.tags && entry.tags.length > 0 && (
                <Flex gap={4} wrap="wrap" style={{ marginTop: 12 }}>
                  {entry.tags.map((t) => (
                    <TagChip key={t.name} tag={t.name} onTagSelect={onTagSelect} />
                  ))}
                </Flex>
              )}
            </Card>
          </EntryLink>
        </Col>
      ))}
    </Row>
  );
}

export default function EntryList({
  entries,
  viewMode,
  authorLinks = false,
  onTagSelect,
  onEntryClick,
}: Props) {
  const shared = { entries, authorLinks, onTagSelect, onEntryClick };
  if (viewMode === "compact") return <EntryListCompact {...shared} />;
  if (viewMode === "comfortable") return <EntryListComfortable {...shared} />;
  return <EntryListGrid {...shared} />;
}
