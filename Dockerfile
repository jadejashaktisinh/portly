FROM golang:1.26-alpine AS relay-build
WORKDIR /src
COPY relay-server/go.mod relay-server/go.sum ./
RUN go mod download
COPY relay-server/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/relay .

FROM node:22-alpine AS api-build
WORKDIR /app
COPY node-backend/package.json node-backend/package-lock.json ./
RUN npm ci
COPY node-backend/ ./
ENV DATABASE_URL=postgresql://placeholder:placeholder@localhost:5432/placeholder
RUN npx prisma generate && npm run build

FROM caddy:2-alpine AS caddy-bin

FROM node:22-alpine
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=api-build /app/node_modules ./node_modules
COPY --from=api-build /app/package.json ./package.json
COPY --from=api-build /app/dist ./dist
COPY --from=api-build /app/prisma ./prisma
COPY --from=api-build /app/prisma.config.ts ./prisma.config.ts

COPY --from=relay-build /out/relay /usr/local/bin/relay
COPY --from=caddy-bin /usr/bin/caddy /usr/local/bin/caddy

COPY docker/Caddyfile /etc/caddy/Caddyfile
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENV PORT=10000
EXPOSE 10000
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
