# Build the application from source
FROM golang:latest AS build-stage

WORKDIR /app

COPY . ./
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /screeps-watcher

# Deploy the application binary into a lean image
# FROM gcr.io/distroless/base-debian11 AS build-release-stage
FROM golang:latest AS build-release-stage

WORKDIR /

COPY config.yaml config.yaml
COPY --from=build-stage /screeps-watcher /screeps-watcher

EXPOSE 2112

#USER nonroot:nonroot

ENTRYPOINT ["/screeps-watcher", "watch", "--config", "/etc/screeps/config.yaml"]
