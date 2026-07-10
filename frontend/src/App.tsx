import { App as AntApp, ConfigProvider } from "antd";
import { Suspense, lazy } from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import ErrorBoundary from "./components/ErrorBoundary";
import LoadingCenter from "./components/common/LoadingCenter";
import { PrefsProvider } from "./context/PrefsContext";
import { ThemeProvider, useThemeMode } from "./context/ThemeContext";
import AppLayout from "./layouts/AppLayout";
import { getThemeConfig } from "./theme";
import { usePageTitle } from "./hooks/usePageTitle";
import Home from "./pages/Home";
import Browse from "./pages/Browse";
import Authors from "./pages/Authors";

const AuthorPage = lazy(() => import("./pages/Author"));
const Platforms = lazy(() => import("./pages/Platforms"));
const EntryPage = lazy(() => import("./pages/Entry"));

function AppRoutes() {
  usePageTitle();

  return (
    <BrowserRouter>
      <Suspense fallback={<LoadingCenter />}>
        <Routes>
          <Route element={<AppLayout />}>
            <Route index element={<Home />} />
            <Route path="browse" element={<Browse />} />
            <Route path="authors" element={<Authors />} />
            <Route path="authors/:dir" element={<AuthorPage />} />
            <Route path="platforms" element={<Platforms />} />
            <Route path="api/*" element={<Navigate to="/" replace />} />
            <Route path="*" element={<EntryPage />} />
          </Route>
        </Routes>
      </Suspense>
    </BrowserRouter>
  );
}

function ThemedApp() {
  const { mode } = useThemeMode();

  return (
    <ConfigProvider theme={getThemeConfig(mode)}>
      <AntApp>
        <PrefsProvider>
          <AppRoutes />
        </PrefsProvider>
      </AntApp>
    </ConfigProvider>
  );
}

export default function App() {
  return (
    <ThemeProvider>
      <ErrorBoundary>
        <ThemedApp />
      </ErrorBoundary>
    </ThemeProvider>
  );
}
