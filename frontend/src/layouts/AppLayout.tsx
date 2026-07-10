import { Flex, Layout, Typography } from "antd";
import { Link as RouterLink, Outlet, useLocation } from "react-router-dom";
import AppMobileNav from "../components/AppMobileNav";
import AppNavMenu from "../components/AppNavMenu";
import BrandMark from "../components/BrandMark";
import SearchBar from "../components/SearchBar";
import SortMenu from "../components/SortMenu";
import ViewSettingsMenu from "../components/ViewSettingsMenu";

const { Header, Content } = Layout;

function navTab(pathname: string, path: string, prefix = false) {
  if (prefix) return pathname === path || pathname.startsWith(`${path}/`);
  return pathname === path;
}

function catalogSortVisible(pathname: string) {
  return pathname === "/browse" || pathname === "/authors" || pathname.startsWith("/authors/");
}

const navItems = [
  { key: "/", label: "Home", to: "/" },
  { key: "/browse", label: "Browse", to: "/browse" },
  { key: "/authors", label: "Authors", to: "/authors" },
  { key: "/platforms", label: "Platforms", to: "/platforms" },
];

export default function AppLayout() {
  const { pathname } = useLocation();

  const activeKey =
    pathname === "/"
      ? "/"
      : navTab(pathname, "/browse")
        ? "/browse"
        : navTab(pathname, "/authors", true)
          ? "/authors"
          : pathname === "/platforms"
            ? "/platforms"
            : "";

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Header className="app-header">
        <div className="app-header__inner app-content-shell">
          <Flex className="app-header__start" align="center" gap={16} style={{ minWidth: 0 }}>
            <div className="app-header__brand">
              <AppMobileNav items={navItems} activeKey={activeKey} />
              <RouterLink to="/" className="app-header__brand-link" aria-label="Warehouse06 home">
                <BrandMark className="app-header__brand-mark app-header__brand-mark--desktop" />
                <Typography.Title level={4} style={{ margin: 0, fontWeight: 800, flexShrink: 0 }}>
                  Warehouse<span className="app-header__brand-accent">06</span>
                </Typography.Title>
              </RouterLink>
            </div>

            <AppNavMenu items={navItems} activeKey={activeKey} className="app-header__nav app-header__nav--desktop" />
          </Flex>

          <Flex className="app-header__end" align="center" gap={4} style={{ minWidth: 0 }}>
            <SearchBar />
            <Flex className="app-header__tools" align="center" gap={4} style={{ flexShrink: 0 }}>
              {catalogSortVisible(pathname) && <SortMenu />}
              <ViewSettingsMenu />
            </Flex>
          </Flex>
        </div>
      </Header>

      <Content className="app-content-shell" style={{ paddingBlock: 32 }}>
        <Outlet />
      </Content>
    </Layout>
  );
}
