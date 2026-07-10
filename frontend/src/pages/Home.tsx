import { Alert, Card, Col, Flex, Row, Tag as AntTag, Typography } from "antd";
import { useMemo } from "react";
import { Link as RouterLink, useLocation, useNavigate } from "react-router-dom";
import { browsePath } from "../lib/browse";
import EntryList from "../components/catalog/EntryList";
import LoadingCenter from "../components/common/LoadingCenter";
import { usePrefs } from "../context/PrefsContext";
import {
  useAuthors,
  useLatestEntries,
  usePlatforms,
  useSyncStatus,
  useTags,
} from "../hooks/useApiQueries";
import { useListRestore } from "../hooks/useListRestore";
import { usePageTitle } from "../hooks/usePageTitle";
import { saveListOrigin, type ListOrigin } from "../lib/listRestore";
import type { Author, Tag } from "../api";

const dateTimeFmt = new Intl.DateTimeFormat(undefined, {
  dateStyle: "medium",
  timeStyle: "short",
});

function shortHash(hash: string) {
  return hash.length > 7 ? hash.slice(0, 7) : hash;
}

function tagByPopularity(a: Tag, b: Tag) {
  const diff = (b.entry_count ?? 0) - (a.entry_count ?? 0);
  return diff !== 0 ? diff : a.name.localeCompare(b.name);
}

function authorByPopularity(a: Author, b: Author) {
  const diff = (b.entry_count ?? 0) - (a.entry_count ?? 0);
  return diff !== 0 ? diff : (a.name || a.directory_name).localeCompare(b.name || b.directory_name);
}

type SectionQuery = { isLoading: boolean; error: Error | null };

// Each home section renders on its own; a slow or failed sidebar query must
// not gate the rest of the page.
function SectionBody({ query, children }: { query: SectionQuery; children: React.ReactNode }) {
  if (query.isLoading) return <LoadingCenter />;
  if (query.error) return <Alert type="error" message={query.error.message} showIcon />;
  return <>{children}</>;
}

function SyncMetaLine() {
  const { data: status } = useSyncStatus();
  if (!status) return null;

  if (status.syncing) {
    return (
      <Typography.Text type="secondary" style={{ fontSize: 13 }}>
        Syncing archive…
      </Typography.Text>
    );
  }

  if (!status.last_synced_at) return null;

  const syncedLabel = dateTimeFmt.format(new Date(status.last_synced_at));
  const commit = status.storage_commit;

  return (
    <Typography.Text type="secondary" style={{ fontSize: 13 }}>
      Archive synced {syncedLabel}
      {commit ? (
        <>
          {" "}
          · storage at{" "}
          <Typography.Text code title={commit.hash}>
            {shortHash(commit.hash)}
          </Typography.Text>{" "}
          ({dateTimeFmt.format(new Date(commit.committed_at))})
          {commit.subject ? (
            <>
              {" "}
              — <Typography.Text italic>{commit.subject}</Typography.Text>
            </>
          ) : null}
        </>
      ) : null}
    </Typography.Text>
  );
}

export default function Home() {
  const { viewMode } = usePrefs();
  const location = useLocation();
  const navigate = useNavigate();
  usePageTitle("Home");
  useListRestore();

  const entriesQuery = useLatestEntries();
  const tagsQuery = useTags();
  const authorsQuery = useAuthors();
  const platformsQuery = usePlatforms();

  const entries = entriesQuery.data?.items ?? [];
  const tags = useMemo(() => [...(tagsQuery.data ?? [])].sort(tagByPopularity), [tagsQuery.data]);
  const authors = useMemo(
    () => [...(authorsQuery.data ?? [])].sort(authorByPopularity),
    [authorsQuery.data],
  );
  const platforms = platformsQuery.data ?? [];

  function handleEntryClick(entryPath: string) {
    const origin: ListOrigin = {
      pathname: location.pathname,
      search: location.search,
      anchorPageIndex: 0,
      anchorEntryPath: entryPath,
      scrollY: window.scrollY,
    };
    saveListOrigin(origin);
    navigate(`/${entryPath}`, { state: { fromList: origin } });
  }

  return (
    <Flex vertical gap={40}>
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={8}>
          <Card title="Platforms">
            <SectionBody query={platformsQuery}>
              <Flex gap={8} wrap="wrap">
                {platforms.map((platform) => (
                  <RouterLink key={platform.path} to={browsePath({ platform: platform.path })}>
                    <AntTag style={{ cursor: "pointer" }}>{platform.name}</AntTag>
                  </RouterLink>
                ))}
              </Flex>
            </SectionBody>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="Popular tags">
            <SectionBody query={tagsQuery}>
              <Flex gap={8} wrap="wrap">
                {tags.slice(0, 20).map((tag) => (
                  <RouterLink key={tag.id} to={browsePath({ tag: tag.name })}>
                    <AntTag color="blue" style={{ cursor: "pointer" }}>
                      {tag.name}
                    </AntTag>
                  </RouterLink>
                ))}
              </Flex>
            </SectionBody>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="Authors">
            <SectionBody query={authorsQuery}>
              <Flex gap={8} wrap="wrap">
                {authors.slice(0, 14).map((author) => (
                  <RouterLink key={author.id} to={`/authors/${author.directory_name}`}>
                    <AntTag style={{ cursor: "pointer" }}>
                      {author.entry_count != null
                        ? `${author.name || author.directory_name} (${author.entry_count})`
                        : author.name || author.directory_name}
                    </AntTag>
                  </RouterLink>
                ))}
              </Flex>
            </SectionBody>
          </Card>
        </Col>
      </Row>

      <div>
        <Flex justify="space-between" align="flex-end" style={{ marginBottom: 16 }}>
          <div>
            <Typography.Text type="secondary" style={{ textTransform: "uppercase", fontSize: 12 }}>
              Recently added
            </Typography.Text>
            <Typography.Title level={2} style={{ margin: "4px 0 0", fontWeight: 800 }}>
              Latest from the archive
            </Typography.Title>
          </div>
          <RouterLink to="/browse" style={{ fontWeight: 600 }}>
            View all
          </RouterLink>
        </Flex>
        <SectionBody query={entriesQuery}>
          <EntryList entries={entries} viewMode={viewMode} onEntryClick={handleEntryClick} />
        </SectionBody>
        <SyncMetaLine />
      </div>
    </Flex>
  );
}
