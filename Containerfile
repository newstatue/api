FROM docker.io/library/ubuntu:24.04

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates libvips42t64 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY api /app/api
RUN chmod +x /app/api

EXPOSE 8080

CMD ["/app/api"]
