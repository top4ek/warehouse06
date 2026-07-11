import { SortAscendingOutlined } from "@ant-design/icons";
import { Button, Dropdown } from "antd";
import type { MenuProps } from "antd";
import {
  SORT_FIELD_LABELS,
  SORT_ORDER_LABELS,
  usePrefs,
  type SortField,
  type SortOrder,
} from "../context/PrefsContext";
import { useHeaderDropdownPanel } from "../hooks/useHeaderDropdownPanel";

const fields: SortField[] = ["date", "created_at", "name"];
const orders: SortOrder[] = ["asc", "desc"];

export default function SortMenu() {
  const { open, onOpenChange, overlayStyle, menuStyle } = useHeaderDropdownPanel();
  const { sortField, sortOrder, setSortField, setSortOrder } = usePrefs();

  const items: MenuProps["items"] = [
    { type: "group", label: "Sort by" },
    ...fields.map((field) => ({
      key: `field-${field}`,
      label: SORT_FIELD_LABELS[field],
      onClick: () => setSortField(field),
    })),
    { type: "divider" },
    { type: "group", label: "Order" },
    ...orders.map((order) => ({
      key: `order-${order}`,
      label: SORT_ORDER_LABELS[order],
      onClick: () => setSortOrder(order),
    })),
  ];

  return (
    <Dropdown
      open={open}
      onOpenChange={onOpenChange}
      styles={{ root: overlayStyle }}
      menu={{
        items,
        selectedKeys: [`field-${sortField}`, `order-${sortOrder}`],
        style: menuStyle,
      }}
      trigger={["click"]}
      placement="bottomRight"
    >
      <Button
        type="text"
        icon={<SortAscendingOutlined />}
        aria-label="Sort entries"
        className="app-header__tool-btn"
      />
    </Dropdown>
  );
}
