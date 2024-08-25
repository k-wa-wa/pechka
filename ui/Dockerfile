FROM node:21.7.3-bookworm-slim as builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY .env.production eslint.config.js index.html postcss.config.js tailwind.config.js \
  tsconfig.app.json tsconfig.json tsconfig.node.json vite.config.ts ./
COPY src/ ./src/

RUN npm run build


FROM nginx:1.27.1-alpine-slim as runner

COPY server.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
