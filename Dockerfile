FROM node:24-alpine AS ui
WORKDIR /ui
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend ./
RUN npm run build

FROM golang:1.27.1-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /email-service ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /email-service /app/email-service
COPY --from=ui /ui/dist /app/web/dist
ENV WEB_DIST=/app/web/dist
USER app
EXPOSE 8080
ENTRYPOINT ["/app/email-service"]
