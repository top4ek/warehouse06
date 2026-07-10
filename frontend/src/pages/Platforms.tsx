import { Alert, Button, Card, Col, Flex, Row, Typography } from "antd";
import { Link as RouterLink } from "react-router-dom";
import { storageUrl } from "../api";
import { browsePath } from "../lib/browse";
import LoadingCenter from "../components/common/LoadingCenter";
import { usePlatforms } from "../hooks/useApiQueries";
import { usePageTitle } from "../hooks/usePageTitle";
import { firstImageSrc, resolveStorageImage } from "../lib/html";

export default function Platforms() {
  const { data: platforms = [], isLoading, error } = usePlatforms();
  usePageTitle("Platforms");

  if (isLoading) return <LoadingCenter />;
  if (error) return <Alert type="error" message={error.message} showIcon />;

  return (
    <Flex vertical gap={24}>
      <div>
        <Typography.Text type="secondary" style={{ textTransform: "uppercase", fontSize: 12 }}>
          Hardware
        </Typography.Text>
        <Typography.Title level={2} style={{ margin: "4px 0 0", fontWeight: 800 }}>
          Platforms
        </Typography.Title>
      </div>

      <Row gutter={[16, 16]}>
        {platforms.map((platform) => {
          const imgSrc = firstImageSrc(platform.content_html);
          const imgUrl = imgSrc
            ? imgSrc.startsWith("http")
              ? imgSrc
              : storageUrl(resolveStorageImage(imgSrc, platform.path))
            : null;

          return (
            <Col key={platform.path} xs={24} md={12}>
              <Card
                cover={
                  imgUrl ? (
                    <img src={imgUrl} alt="" style={{ height: 160, objectFit: "cover" }} />
                  ) : undefined
                }
              >
                <Flex justify="space-between" align="flex-start" gap={16}>
                  <div>
                    <RouterLink to={`/${platform.path}`} style={{ color: "inherit", textDecoration: "none" }}>
                      <Typography.Title level={4} style={{ margin: 0, fontWeight: 800 }}>
                        {platform.name || platform.path}
                      </Typography.Title>
                    </RouterLink>
                    <Typography.Text type="secondary" style={{ display: "block", marginTop: 4 }}>
                      {platform.entry_count} entries
                    </Typography.Text>
                  </div>
                  <RouterLink to={browsePath({ platform: platform.path })}>
                    <Button type="primary">Browse</Button>
                  </RouterLink>
                </Flex>
                {platform.description && (
                  <Typography.Paragraph
                    type="secondary"
                    ellipsis={{ rows: 3 }}
                    style={{ marginTop: 16, marginBottom: 0 }}
                  >
                    {platform.description}
                  </Typography.Paragraph>
                )}
              </Card>
            </Col>
          );
        })}
      </Row>
    </Flex>
  );
}
