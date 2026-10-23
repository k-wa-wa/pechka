import type { StorybookConfig } from "@storybook/nextjs-vite";
import { fileURLToPath } from "url";

const config: StorybookConfig = {
  stories: ["../components/**/*.stories.@(js|jsx|mjs|ts|tsx)"],
  addons: ["@storybook/addon-docs", "msw-storybook-addon"],
  framework: "@storybook/nextjs-vite",
  staticDirs: ["../public"],
  async viteFinal(viteConfig) {
    viteConfig.resolve ??= {};
    viteConfig.resolve.alias = {
      ...viteConfig.resolve.alias,
      "@": fileURLToPath(new URL("..", import.meta.url)),
    };
    return viteConfig;
  },
};
export default config;
