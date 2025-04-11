FROM node:23-bullseye-slim as base

ENV NODE_ENV production

RUN apt-get update && apt-get install -y openssl

FROM base as deps

ENV NODE_ENV development
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

FROM base as build

WORKDIR /app

COPY --from=deps /app/node_modules /app/node_modules

COPY public ./public
COPY package.json postcss.config.cjs tsconfig.json vite.config.ts ./
COPY app ./app
COPY src ./src
RUN npm run build

FROM base

WORKDIR /app

COPY --from=build /app/node_modules /app/node_modules

COPY --from=build /app/build /app/build
COPY --from=build /app/public /app/public

CMD ["npm", "start"]
