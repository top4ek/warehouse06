import { ArrowLeftOutlined, PlayCircleOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Col, Flex, Row, Tag, Typography, theme } from "antd";
import { useState } from "react";
import { Link as RouterLink, useLocation, useNavigate, useParams } from "react-router-dom";
import { type FileItem } from "../api";
import { browsePath } from "../lib/browse";
import {
  isReservedEntryPath,
  parsePlayHash,
  parsePlaySearch,
  parsePlaySplat,
} from "../lib/playRoute";
import { formatAddedAt } from "../lib/format";
import { listOriginLabel, readListOrigin, type ListOrigin } from "../lib/listRestore";
import { sanitizeHtml } from "../lib/sanitizeHtml";
import { parseYouTubeId } from "../lib/youtube";
import LinkList from "../components/common/LinkList";
import LoadingCenter from "../components/common/LoadingCenter";
import EmulatorDialog from "../components/EmulatorDialog";
import EntrySidebarResources from "../components/EntrySidebarResources";
import VideoDialog from "../components/VideoDialog";
import { useEntry } from "../hooks/useApiQueries";
import { usePageTitle } from "../hooks/usePageTitle";

const PLAYABLE_EXT = [".rom", ".fdd", ".zip", ".com", ".bin", ".r0m"];

function isPlayable(file: FileItem) {
  return PLAYABLE_EXT.some((ext) => file.filename.toLowerCase().endsWith(ext));
}

function SidebarSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card size="small" style={{ background: "var(--ant-color-fill-quaternary)" }}>
      <Typography.Text type="secondary" style={{ textTransform: "uppercase", fontSize: 12 }}>
        {title}
      </Typography.Text>
      {children}
    </Card>
  );
}

export default function EntryPage() {
  const { hash, search } = useLocation();
  const { "*": pathParam } = useParams();
  const path = pathParam ?? "";
  const playSplat = parsePlaySplat(path);
  const entryPath = playSplat?.entryPath ?? path;
  const playFilename = playSplat?.filename ?? parsePlaySearch(search) ?? parsePlayHash(hash);

  // Keyed by entry path so per-entry UI state (dialogs) resets on navigation
  // instead of being cleaned up by effects.
  return <EntryView key={entryPath} entryPath={entryPath} playFilename={playFilename} />;
}

function EntryView({ entryPath, playFilename }: { entryPath: string; playFilename: string | null }) {
  const { token } = theme.useToken();
  const location = useLocation();
  const navigate = useNavigate();
  const reserved = isReservedEntryPath(entryPath);
  const {
    data: entry,
    isLoading: loading,
    error: queryError,
  } = useEntry(entryPath, Boolean(entryPath) && !reserved);
  const error = reserved ? "Not found" : (queryError?.message ?? "");
  const [videoOpen, setVideoOpen] = useState(false);

  usePageTitle(entry?.name || entry?.path || "Entry");

  // The emulator dialog is driven entirely by the URL: a playable file
  // named by the play route opens it; closing navigates the play part away.
  const playFile =
    playFilename && entry
      ? entry.files?.find((f) => f.filename === playFilename && isPlayable(f))
      : undefined;
  const emulatorOpen = Boolean(playFile);
  const currentRom = playFile && entry ? playFile.filepath || `${entry.path}/${playFile.filename}` : "";

  function closeEmulator() {
    navigate({ pathname: `/${entryPath}`, search: "", hash: "" }, { replace: true });
  }

  if (loading) return <LoadingCenter />;
  if (error) return <Alert type="error" title={error} showIcon />;
  if (!entry) return null;

  const youtubeId = parseYouTubeId(entry.youtube);
  const addedLabel = formatAddedAt(entry.created_at);
  const fromList = (location.state as { fromList?: ListOrigin } | null)?.fromList ?? readListOrigin();

  function goBackToList() {
    if (!fromList) return;
    navigate(
      { pathname: fromList.pathname, search: fromList.search },
      { state: { restore: fromList } },
    );
  }

  return (
    <Flex vertical gap={32}>
      <Card>
        <Row gutter={[32, 32]}>
          <Col xs={24} md={16}>
            {fromList && (
              <Button
                type="link"
                icon={<ArrowLeftOutlined />}
                onClick={goBackToList}
                style={{ padding: 0, marginBottom: 8, height: "auto" }}
              >
                {listOriginLabel(fromList)}
              </Button>
            )}
            <Flex vertical gap={8} style={{ paddingBottom: 24, borderBottom: "1px solid var(--ant-color-border)" }}>
              <Typography.Text type="secondary" style={{ textTransform: "uppercase", fontSize: 12 }}>
                {entry.platform || "Entry"}
              </Typography.Text>
              <Typography.Title level={2} style={{ margin: 0, fontWeight: 800 }}>
                {entry.name || entry.path}
              </Typography.Title>
              {(entry.date || addedLabel) && (
                <Typography.Text type="secondary">
                  {[entry.date, addedLabel && `Added ${addedLabel}`].filter(Boolean).join(" · ")}
                </Typography.Text>
              )}
            </Flex>
            <div
              className="content-html"
              style={{ marginTop: 24 }}
              dangerouslySetInnerHTML={{ __html: sanitizeHtml(entry.content_html ?? "") }}
            />
          </Col>

          <Col xs={24} md={8}>
            <Flex vertical gap={16}>
              {entry.authors && entry.authors.length > 0 && (
                <SidebarSection title="Authors">
                  <div style={{ marginTop: 8 }}>
                    <LinkList
                      items={entry.authors.map((a) => ({
                        key: a.directory_name,
                        label: a.name || a.directory_name,
                        to: `/authors/${a.directory_name}`,
                      }))}
                    />
                  </div>
                </SidebarSection>
              )}

              {entry.tags && entry.tags.length > 0 && (
                <SidebarSection title="Tags">
                  <Flex gap={8} wrap="wrap" style={{ marginTop: 8 }}>
                    {entry.tags.map((tag) => (
                      <RouterLink key={tag.name} to={browsePath({ tag: tag.name })}>
                        <Tag style={{ cursor: "pointer" }}>{tag.name}</Tag>
                      </RouterLink>
                    ))}
                  </Flex>
                </SidebarSection>
              )}

              {youtubeId && (
                <SidebarSection title="Video">
                  <Card
                    hoverable
                    size="small"
                    style={{ marginTop: 8, cursor: "pointer" }}
                    onClick={() => setVideoOpen(true)}
                    cover={
                      <img
                        src={`https://img.youtube.com/vi/${youtubeId}/mqdefault.jpg`}
                        alt=""
                        loading="lazy"
                        decoding="async"
                        style={{ aspectRatio: "16/9", objectFit: "cover" }}
                      />
                    }
                  >
                    <Flex align="center" justify="center" gap={4} style={{ padding: "8px 0", background: token.colorBgSpotlight, color: token.colorTextLightSolid }}>
                      <PlayCircleOutlined />
                      <Typography.Text style={{ color: token.colorTextLightSolid, fontSize: 12, fontWeight: 500 }}>
                        Watch
                      </Typography.Text>
                    </Flex>
                  </Card>
                </SidebarSection>
              )}

              {entry.require && entry.require.length > 0 && (
                <SidebarSection title="Requires">
                  <div style={{ marginTop: 8 }}>
                    <LinkList
                      items={entry.require.map((req) => ({
                        key: req,
                        label: req,
                        to: `/${req}`,
                      }))}
                    />
                  </div>
                </SidebarSection>
              )}

              <EntrySidebarResources
                directories={entry.directories ?? []}
                files={entry.files ?? []}
                entryPath={entry.path}
                isPlayable={isPlayable}
              />
            </Flex>
          </Col>
        </Row>
      </Card>

      <EmulatorDialog open={emulatorOpen} romUrl={currentRom} onClose={closeEmulator} />
      <VideoDialog open={videoOpen} youtubeId={youtubeId} onClose={() => setVideoOpen(false)} />
    </Flex>
  );
}
