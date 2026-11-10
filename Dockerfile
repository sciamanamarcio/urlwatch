FROM golang:1.25-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/urlwatch ./cmd/urlwatch

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
USER app
COPY --from=build /out/urlwatch /usr/local/bin/urlwatch
EXPOSE 8080
ENTRYPOINT ["urlwatch"]
CMD ["serve", "-addr", ":8080"]
