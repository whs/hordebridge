FROM golang:1.26 AS build
COPY . /src/
RUN cd /src/ && go generate ./... && go build -o hordebridge

FROM debian:trixie-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /src/hordebridge /opt/hordebridge
CMD ["/opt/hordebridge"]
