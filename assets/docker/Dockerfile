# shared deps
ARG go_version=1.19.0
FROM "golang:${go_version}-bullseye" as base
RUN apt-get -y update
RUN apt-get -y --no-install-recommends install autoconf automake make \
    libtool liblz4-dev libuv1-dev libsqlite3-dev

# raft & dqlite
FROM base as dqlite
# raft
ARG raft_version=v0.15.0
RUN git clone --branch "${raft_version}" --depth 1 https://github.com/canonical/raft.git /tmp/raft
WORKDIR /tmp/raft
RUN autoreconf -i
RUN ./configure
RUN make
RUN make install
# dqlite
ARG dqlite_version=v1.11.1
RUN git clone --branch "${dqlite_version}" --depth 1 https://github.com/canonical/dqlite.git /tmp/dqlite
WORKDIR /tmp/dqlite
RUN autoreconf -i
RUN ./configure
RUN make
RUN make install

# sqlite3
FROM base as sqlite3
ARG sqlite_version=3390200
RUN curl -fsSL "https://www.sqlite.org/2022/sqlite-autoconf-${sqlite_version}.tar.gz" | tar -xz -C /tmp
WORKDIR "/tmp/sqlite-autoconf-${sqlite_version}"
RUN autoreconf -i
RUN ./configure --enable-fts5
RUN make
RUN make install

# consolidate
FROM dqlite as builder-base
COPY --from=sqlite3 /usr/local/lib /usr/local/lib
RUN ldconfig
ENV CGO_LDFLAGS_ALLOW="-Wl,-z,now"

# build go source
FROM builder-base as builder
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -a -tags libsqlite3 -o server ./cmd/cluster/

# final image
FROM debian:bullseye-slim
RUN apt -y update && apt -y install libuv1-dev
COPY --from=builder /usr/local/lib /usr/local/lib
RUN ldconfig
ARG uid=65532
RUN useradd --no-create-home --system --shell /bin/false --uid "$uid" app
USER "$uid"
WORKDIR /opt/app
COPY --from=builder /workspace/server ./server
ENTRYPOINT [ "./server" ]
CMD ["-v", "2"]

