FROM postgres:16 AS builder

COPY --from=golang:1.25-bookworm /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates build-essential postgresql-server-dev-16 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make -C ext/formql install

FROM postgres:16

COPY --from=builder /usr/lib/postgresql/16/lib/formql.so /usr/lib/postgresql/16/lib/formql.so
COPY --from=builder /usr/lib/postgresql/16/lib/libformql_engine.so /usr/lib/postgresql/16/lib/libformql_engine.so
COPY --from=builder /usr/share/postgresql/16/extension/formql.control /usr/share/postgresql/16/extension/formql.control
COPY --from=builder /usr/share/postgresql/16/extension/formql--0.1.0.sql /usr/share/postgresql/16/extension/formql--0.1.0.sql
