FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# Empty base URL = relative /api/v1 (same-origin through Caddy in prod).
ARG VITE_API_BASE_URL=
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

# Realtime WebSocket switch; "false" degrades the app to polling (e.g. test env without WS).
ARG VITE_REALTIME_ENABLED=true
ENV VITE_REALTIME_ENABLED=$VITE_REALTIME_ENABLED

RUN npm run build

FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
