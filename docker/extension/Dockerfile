FROM postgres:16

RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential postgresql-server-dev-16 \
    && rm -rf /var/lib/apt/lists/*

COPY ext/formql /tmp/formql
RUN make -C /tmp/formql && make -C /tmp/formql install
