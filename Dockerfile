FROM golang:1.25-alpine AS build
ARG SERVICE
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /out/app ./cmd/${SERVICE}

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 kmap
USER kmap
COPY --from=build /out/app /app
ENTRYPOINT ["/app"]
