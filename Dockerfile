# Stage 1
# Run the latest Go
FROM golang:alpine AS build

ARG VERSION
RUN apk add --update --no-cache git make upx
WORKDIR /app/
COPY go.mod go.sum /app/
RUN go mod download
RUN go mod verify
COPY . /app/

# Stage 2
# Build Carbon and pack
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X github.com/Zentro/carbon/system.Version=$VERSION" \
    -v \
    -tags=jsoniter \
    -trimpath \
    -o carbon \
    carbon.go
RUN upx carbon
RUN echo "ID=\"distroless\"" > /etc/os-release

# Stage 3
# ref: https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static:latest
COPY --from=build /etc/os-release /etc/os-release
COPY --from=build /app/carbon /usr/bin/
CMD [ "/usr/bin/carbon", "--config", "/etc/carbon/config.yml" ]
EXPOSE 8080
