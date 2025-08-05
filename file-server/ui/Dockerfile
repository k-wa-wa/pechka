FROM node:24-bookworm-slim as base

ENV NODE_ENV=production

RUN apt-get update && apt-get install -y openssl

FROM base as deps

ENV NODE_ENV=development
WORKDIR /app

COPY package.json ./
RUN npm i

FROM base as build

WORKDIR /app

COPY --from=deps /app/node_modules /app/node_modules
COPY --from=deps /app/package.json /app/package.json

COPY package.json postcss.config.cjs tsconfig.json vite.config.ts ./
COPY app ./app
COPY src ./src
RUN npm run build

FROM base as runner

WORKDIR /app

COPY --from=build /app/node_modules /app/node_modules
COPY --from=build /app/package.json /app/package.json

COPY --from=build /app/build /app/build
COPY server.js ./

CMD ["npm", "start"]
