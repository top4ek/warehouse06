import { Button, Drawer, theme } from "antd";
import { useMemo, useState } from "react";
import { Link as RouterLink, useLocation } from "react-router-dom";
import type { NavItem } from "./AppNavMenu";
import BrandMark from "./BrandMark";

type AppMobileNavProps = {
  items: NavItem[];
  activeKey: string;
};

export default function AppMobileNav({ items, activeKey }: AppMobileNavProps) {
  const { token } = theme.useToken();
  const [open, setOpen] = useState(false);
  const { pathname } = useLocation();

  const panelStyle = useMemo(
    () => ({
      backgroundColor: token.colorBgContainer,
      color: token.colorText,
    }),
    [token.colorBgContainer, token.colorText],
  );

  // Close the drawer whenever the route changes (adjust-during-render,
  // https://react.dev/learn/you-might-not-need-an-effect).
  const [lastPathname, setLastPathname] = useState(pathname);
  if (pathname !== lastPathname) {
    setLastPathname(pathname);
    setOpen(false);
  }

  return (
    <>
      <Button
        type="text"
        className="app-header__menu-btn"
        aria-label="Main menu"
        aria-expanded={open}
        aria-controls="app-mobile-nav"
        onClick={() => setOpen(true)}
      >
        <BrandMark className="app-header__brand-mark" width={28} height={28} />
      </Button>
      <Drawer
        id="app-mobile-nav"
        title="Menu"
        placement="left"
        open={open}
        onClose={() => setOpen(false)}
        width={280}
        classNames={{ body: "app-mobile-nav-drawer__body" }}
        styles={{
          wrapper: panelStyle,
          content: panelStyle,
          header: panelStyle,
          body: panelStyle,
        }}
        destroyOnClose
      >
        <nav className="app-mobile-nav" aria-label="Main">
          {items.map((item) => {
            const active = item.key === activeKey;
            return (
              <RouterLink
                key={item.key}
                to={item.to}
                className={`app-mobile-nav__item${active ? " app-mobile-nav__item--active" : ""}`}
                aria-current={active ? "page" : undefined}
                onClick={() => setOpen(false)}
              >
                {item.label}
              </RouterLink>
            );
          })}
        </nav>
      </Drawer>
    </>
  );
}
