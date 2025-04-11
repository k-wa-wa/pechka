import { vitePlugin as remix } from "@remix-run/dev"
import { defineConfig } from "vite"
import path from 'path';

const API_URL = "http://localhost:8000"

export default defineConfig({
  plugins: [remix()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './'),
    },
  }
})
