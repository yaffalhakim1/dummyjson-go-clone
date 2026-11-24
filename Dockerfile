FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=build /server /app/server
COPY data/ /app/data/
EXPOSE 8888
ENV ADDR=:8888 DATA_DIR=/app/data JWT_SECRET=change-me
CMD ["/app/server"]
