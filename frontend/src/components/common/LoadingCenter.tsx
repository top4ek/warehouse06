import { Flex, Spin } from "antd";

export default function LoadingCenter() {
  return (
    <Flex justify="center" style={{ padding: "64px 0" }}>
      <Spin size="large" />
    </Flex>
  );
}
