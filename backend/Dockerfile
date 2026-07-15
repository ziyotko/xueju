FROM golang:1.21.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/xueju-api ./cmd/api

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app && apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/xueju-api ./xueju-api
COPY internal/textfilter/sensitive_words.txt ./internal/textfilter/sensitive_words.txt
USER app
EXPOSE 8080
ENTRYPOINT ["./xueju-api"]
