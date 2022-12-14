## BUILD CONTAINER ##
# The ENV valued (specified by --build-args <ENV>) determines which gobuilder options to use.
ARG ENV
FROM golang:1.19.1 as builder

RUN mkdir /build
WORKDIR /build
# Copy over contents of current dir into /build
ADD . .
WORKDIR /build/cmd/builder

# Makes use of the ENV build-arg.
RUN if [ "$ENV" = "m1-laptop-dev" ] ; then go build -o /build/dist/builder --ldflags="-s -w" -trimpath . ; else GOOS=linux GOARCH=amd64 go build -o /build/dist/builder --ldflags="-s -w" -trimpath . ; fi

WORKDIR /build
# Run the builder and output into builder/dist/otelcorecol
RUN /build/dist/builder --config /build/cmd/otelcorecol/builder-config.yaml --output-path=/build/dist/otelcorecol


## RUN CONTAINER ##
# What will actually get run when the container is deployed.
FROM ubuntu:latest

RUN mkdir /app
COPY --from=builder /build/dist/otelcorecol /app
COPY /build/otc-config.yaml /etc/otel/config.yaml

ENTRYPOINT ["/app/otelcorecol","--config=/etc/otel/config.yaml"]