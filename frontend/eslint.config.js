import js from "@eslint/js";
import tseslint from "typescript-eslint";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";

export default tseslint.config(
  js.configs.recommended,
  ...tseslint.configs.recommended,
  reactHooks.configs.flat["recommended-latest"],
  reactRefresh.configs.vite,
  {
    ignores: ["dist/**", "public/emulator/**", "emulator-src/**"],
  },
  {
    // Context modules intentionally export hooks/constants next to their
    // provider; fast-refresh limitations are acceptable there.
    files: ["src/context/**"],
    rules: {
      "react-refresh/only-export-components": "off",
    },
  },
);
