FROM golang:1.24-bookworm AS build
WORKDIR /app

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./app/gin-server

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=build /app/server /app/server

ENV PORT=8080
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/server"]