FROM node:22-alpine AS build
WORKDIR /app
RUN corepack enable
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm build

FROM node:22-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
RUN addgroup -S kmap && adduser -S -u 10001 -G kmap kmap
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
USER kmap
EXPOSE 3000
CMD ["node", "server.js"]
