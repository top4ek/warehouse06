import { Alert, Empty, Flex, Typography } from "antd";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { useCatalog, CATALOG_PAGE_SIZE } from "../../hooks/useCatalog";
import { useBidirectionalScroll } from "../../hooks/useBidirectionalScroll";
import { useListRestore } from "../../hooks/useListRestore";
import { browsePath } from "../../lib/browse";
import { saveListOrigin, type ListOrigin } from "../../lib/listRestore";
import { usePrefs } from "../../context/PrefsContext";
import ScrollToTopFab from "../ScrollToTopFab";
import EntryList from "./EntryList";
import LoadingCenter from "../common/LoadingCenter";

type Props = {
  mode?: "browse" | "authors";
  fixedAuthor?: string;
};

export default function EntryCatalog({ mode = "browse", fixedAuthor = "" }: Props) {
  const navigate = useNavigate();
  const location = useLocation();
  const { sortField, sortOrder, viewMode } = usePrefs();
  const [searchParams] = useSearchParams();
  const searchKey = searchParams.toString();

  const initialRestore = useListRestore();

  const {
    entries,
    pageRangeStart,
    pageRangeEnd,
    loading,
    loadingMore,
    loadingPrev,
    error,
    hasMore,
    hasPrev,
    loadNext,
    loadPrev,
  } = useCatalog({
    mode,
    fixedAuthor,
    searchKey,
    sortField,
    sortOrder,
    restore: initialRestore,
  });

  const { topRef, bottomRef } = useBidirectionalScroll(loadPrev, loadNext, {
    canLoadPrev: hasPrev && !loading,
    canLoadNext: hasMore && !loading,
    resetKey: `${pageRangeStart}:${pageRangeEnd}`,
  });

  function handleEntryClick(entryPath: string, indexInWindow: number) {
    const origin: ListOrigin = {
      pathname: location.pathname,
      search: location.search,
      anchorPageIndex: pageRangeStart + Math.floor(indexInWindow / CATALOG_PAGE_SIZE),
      anchorEntryPath: entryPath,
      scrollY: window.scrollY,
    };
    saveListOrigin(origin);
    navigate(`/${entryPath}`, { state: { fromList: origin } });
  }

  return (
    // Native scroll anchoring is disabled here: the catalog compensates the
    // scroll position itself when the page window shifts, and the two
    // mechanisms fight each other.
    <section style={{ overflowAnchor: "none" }}>
      <Flex vertical gap={16}>
        {loading && entries.length === 0 ? (
          <LoadingCenter />
        ) : error ? (
          <Alert type="error" message={error} showIcon />
        ) : entries.length > 0 ? (
          <>
            {loadingPrev && (
              <Typography.Text type="secondary" style={{ textAlign: "center", display: "block", padding: "8px 0" }}>
                Loading previous…
              </Typography.Text>
            )}
            <div ref={topRef} style={{ height: 1 }} aria-hidden />
            <EntryList
              entries={entries}
              viewMode={viewMode}
              authorLinks={mode === "authors"}
              onTagSelect={(tag) => navigate(browsePath({ tag }))}
              onEntryClick={mode === "authors" ? undefined : handleEntryClick}
            />
            <div ref={bottomRef} style={{ height: 32 }} aria-hidden />
            {loadingMore && (
              <Typography.Text type="secondary" style={{ textAlign: "center", display: "block", padding: "16px 0" }}>
                Loading more…
              </Typography.Text>
            )}
          </>
        ) : (
          <Empty description="Nothing found." />
        )}
        <ScrollToTopFab />
      </Flex>
    </section>
  );
}
