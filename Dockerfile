# Build stage
FROM golang:1.23 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ncore-auto-seed-checker

# Final image with headless Chrome
FROM chromedp/headless-shell:latest
WORKDIR /app
COPY --from=build /app/ncore-auto-seed-checker ./
ENV CHROME_PATH=/headless-shell/headless-shell
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["./ncore-auto-seed-checker"]
