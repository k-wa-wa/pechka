FROM node:18-alpine AS base

FROM base AS deps

RUN apk add --no-cache libc6-compat
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

FROM base AS runner

ENV NODE_ENV=production
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs
USER nextjs

WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules

COPY public ./public
COPY package.json next.config.ts postcss.config.mjs tailwind.config.ts tsconfig.json ./
COPY app ./app

EXPOSE 3000

CMD [ "/bin/sh", "-c", "npm run build && npm run start" ]
