# Build context is the repository root (the frontend imports
# backend/internal/domain/load_thresholds.json).
FROM node:24-alpine AS build
WORKDIR /repo/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY backend/internal/domain/load_thresholds.json /repo/backend/internal/domain/load_thresholds.json
COPY frontend/ ./
RUN npm run build

FROM nginx:1.29-alpine
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /repo/frontend/dist /usr/share/nginx/html
EXPOSE 8080
