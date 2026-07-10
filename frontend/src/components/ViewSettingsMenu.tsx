import { SettingOutlined } from "@ant-design/icons";
import { Button, Dropdown } from "antd";
import type { MenuProps } from "antd";
import { useThemeMode } from "../context/ThemeContext";
import { usePrefs, type ViewMode } from "../context/PrefsContext";
import { useHeaderDropdownPanel } from "../hooks/useHeaderDropdownPanel";

const modes: { id: ViewMode; label: string }[] = [
  { id: "compact", label: "Small" },
  { id: "comfortable", label: "Medium" },
  { id: "grid", label: "Tiles" },
];

export default function ViewSettingsMenu() {
  const { open, onOpenChange, overlayStyle, menuStyle } = useHeaderDropdownPanel();
  const { viewMode, setViewMode } = usePrefs();
  const { mode, toggleMode } = useThemeMode();

  const items: MenuProps["items"] = [
    { type: "group", label: "View" },
    ...modes.map(({ id, label }) => ({
      key: `view-${id}`,
      label,
      onClick: () => setViewMode(id),
    })),
    { type: "divider" },
    { type: "group", label: "Theme" },
    {
      key: "theme-toggle",
      label: mode === "dark" ? "Day mode" : "Night mode",
      onClick: toggleMode,
    },
  ];

  const selectedKeys = [`view-${viewMode}`];

  return (
    <Dropdown
      open={open}
      onOpenChange={onOpenChange}
      overlayStyle={overlayStyle}
      menu={{ items, selectedKeys, style: menuStyle }}
      trigger={["click"]}
      placement="bottomRight"
    >
      <Button
        type="text"
        icon={<SettingOutlined />}
        aria-label="View and theme settings"
        className="app-header__tool-btn"
      />
    </Dropdown>
  );
}
