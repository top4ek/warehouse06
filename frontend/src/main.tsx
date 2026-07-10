import "@ant-design/v5-patch-for-react-19";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import App from "./App";
import { queryClient } from "./api/queryClient";
import "./index.css";

const rootEl = document.getElementById("app");
if (!rootEl) {
  throw new Error('Missing root element "#app"');
}

createRoot(rootEl).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
);
