FROM node:24-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.27.3-alpine AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN go test ./... && CGO_ENABLED=0 go build -trimpath -o /haojiayuan ./cmd/haojiayuan

FROM alpine:3.22
RUN addgroup -g 10001 app && adduser -D -u 10001 -G app app && mkdir /data && chown app:app /data
COPY --from=backend /haojiayuan /app/haojiayuan
COPY --from=frontend /src/frontend/dist /app/public
USER 10001:10001
ENV LISTEN_ADDR=0.0.0.0:8080 DATA_DIR=/data STATIC_DIR=/app/public
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -q -O /dev/null http://127.0.0.1:8080/readyz || exit 1
ENTRYPOINT ["/app/haojiayuan"]
