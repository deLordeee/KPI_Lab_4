FROM golang:1.24 AS build
WORKDIR /go/src/practice-4

# Спочатку копіюємо файли залежностей та завантажуємо їх
COPY go.mod go.sum ./
RUN go mod download

# Копіюємо весь код та запускаємо тести
COPY . .
RUN go test -v ./...

# Збираємо бінарники
ENV CGO_ENABLED=0
RUN go install ./cmd/...

# ==== Final image ====
FROM alpine:latest
WORKDIR /opt/practice-4

# Створюємо entry script
RUN echo '#!/bin/sh\n\
bin=$1\n\
shift\n\
if [ -z "$bin" ]; then\n\
  echo "binary is not defined"\n\
  exit 1\n\
fi\n\
exec ./"$bin" "$@"' > entry.sh && \
    chmod +x entry.sh

# Копіюємо бінарники з етапу збірки
COPY --from=build /go/bin/* /opt/practice-4/

ENTRYPOINT ["/opt/practice-4/entry.sh"]
CMD ["server"]
