# ---- build stage ----
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/node:20-alpine AS build
WORKDIR /app

# 先 COPY package*.json 走层缓存：依赖不变就不会重跑 npm ci
COPY package*.json ./
RUN npm ci

# 再 COPY 源码并 build
COPY . .
RUN npm run build

# ---- runtime stage ----
FROM swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/nginx:1.27-alpine

# 把构建产物和 nginx 配置烤进镜像
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

# alpine 镜像自带 wget；用首页存活即代表 nginx 起来了
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget -qO- http://localhost/ >/dev/null || exit 1

EXPOSE 80

# nginx:1.27-alpine 默认 CMD 就是 nginx -g "daemon off;"
