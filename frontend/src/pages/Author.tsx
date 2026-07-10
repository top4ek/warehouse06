import { Alert, Card, Flex, Typography } from "antd";
import { useParams } from "react-router-dom";
import EntryCatalog from "../components/catalog/EntryCatalog";
import LoadingCenter from "../components/common/LoadingCenter";
import { useAuthor } from "../hooks/useApiQueries";
import { usePageTitle } from "../hooks/usePageTitle";
import { sanitizeHtml } from "../lib/sanitizeHtml";

export default function AuthorPage() {
  const { dir = "" } = useParams();
  const { data: author, isLoading, error } = useAuthor(dir);
  usePageTitle(author?.name || author?.directory_name || "Author");

  if (isLoading) return <LoadingCenter />;
  if (error) return <Alert type="error" message={error.message} showIcon />;
  if (!author) return null;

  return (
    <Flex vertical gap={32}>
      <Card>
        <Flex vertical gap={8} style={{ padding: "8px 0" }}>
          <Typography.Title level={2} style={{ margin: 0, fontWeight: 800 }}>
            {author.name || author.directory_name}
          </Typography.Title>
          {author.address && (
            <Typography.Text type="secondary">{author.address}</Typography.Text>
          )}
          {author.entry_count != null && (
            <Typography.Text type="secondary">
              {author.entry_count} {author.entry_count === 1 ? "work" : "works"} in the archive
            </Typography.Text>
          )}
        </Flex>
        {author.content_html && (
          <div
            className="content-html"
            style={{ borderTop: "1px solid var(--ant-color-border)", paddingTop: 24, marginTop: 16 }}
            dangerouslySetInnerHTML={{ __html: sanitizeHtml(author.content_html) }}
          />
        )}
      </Card>

      <div>
        <Typography.Title level={4} style={{ fontWeight: 800, marginBottom: 16 }}>
          Works
        </Typography.Title>
        <EntryCatalog mode="browse" fixedAuthor={author.directory_name} />
      </div>
    </Flex>
  );
}
