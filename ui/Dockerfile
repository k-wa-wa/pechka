FROM node:21.7.3-bookworm-slim as builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY .env.production postcss.config.cjs tsconfig.json vite.config.ts ./
COPY public/ ./public/
COPY app/ ./app/

RUN npm run build


FROM node:21.7.3-bookworm-slim as production-deps

WORKDIR /app

COPY --from=builder /app/node_modules /app/node_modules
COPY package.json package-lock.json ./
RUN npm prune --omit=dev


FROM node:21.7.3-bookworm-slim as runner

WORKDIR /app
COPY --from=production-deps /app/node_modules /app/node_modules
COPY --from=builder /app/build /app/build
COPY --from=builder /app/public /app/public
COPY package.json ./

CMD [ "npm", "start" ]
