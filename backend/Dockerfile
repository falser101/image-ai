# ---- build stage ----
# CGO 必须开启：mattn/go-sqlite3 是 CGO 库，关掉会 link 失败
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:1.23-bookworm AS build
WORKDIR /src

# Go 模块代理换成国内源（goproxy.cn），direct 表示私有 module 走直连
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=off

# 先 COPY go.mod/go.sum 走 Docker 层缓存：依赖没变就不会重跑 mod download
COPY go.mod go.sum ./
RUN go mod download

# 再 COPY 源码
COPY . .

# -trimpath 去本地路径；-s -w 剥调试信息
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server .

# ---- runtime stage ----
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates wget \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=build /out/server /app/server

# 容器内默认值；实际值由 docker-compose 注入
ENV PORT=8080 \
    DB_PATH=/app/data/data.db \
    UPLOAD_DIR=/app/data/uploads

EXPOSE 8080

# /api/health 是公开路由（main.go:49），直接 wget 探活
HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/server"]
