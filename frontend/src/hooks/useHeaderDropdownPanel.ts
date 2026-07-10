import { theme } from "antd";
import type { CSSProperties } from "react";
import { useHeaderDropdown } from "./useHeaderDropdown";

export function useHeaderDropdownPanel() {
  const { token } = theme.useToken();
  const dropdown = useHeaderDropdown();

  const panelStyle: CSSProperties = {
    backgroundColor: token.colorBgContainer,
    color: token.colorText,
    boxShadow: token.boxShadowSecondary,
    borderRadius: token.borderRadiusLG,
    border: `1px solid ${token.colorBorderSecondary}`,
  };

  return {
    open: dropdown.open,
    onOpenChange: dropdown.onOpenChange,
    overlayStyle: panelStyle,
    menuStyle: panelStyle,
  };
}
