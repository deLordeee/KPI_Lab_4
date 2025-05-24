FROM golang:1.24 AS build
WORKDIR /go/src/practice-4
COPY . .
RUN go test ./...
ENV CGO_ENABLED=0
RUN go install ./cmd/...

# ==== Final image ====
FROM alpine:latest
WORKDIR /opt/practice-4

# Create entry script directly in container (no line ending issues)
RUN echo '#!/bin/sh' > /opt/practice-4/entry.sh && \
    echo 'bin=$1' >> /opt/practice-4/entry.sh && \
    echo 'shift' >> /opt/practice-4/entry.sh && \
    echo 'if [ -z "$bin" ]; then' >> /opt/practice-4/entry.sh && \
    echo '  echo "binary is not defined"' >> /opt/practice-4/entry.sh && \
    echo '  exit 1' >> /opt/practice-4/entry.sh && \
    echo 'fi' >> /opt/practice-4/entry.sh && \
    echo 'exec ./"$bin" $@' >> /opt/practice-4/entry.sh && \
    chmod +x /opt/practice-4/entry.sh

COPY --from=build /go/bin/* /opt/practice-4/
RUN ls -la /opt/practice-4

ENTRYPOINT ["/opt/practice-4/entry.sh"]
CMD ["server"]
