## 使用和 go.mod 文件中相通的 go 版本 编译go文件
## 通过使用 AS 来 进行多阶段构建
FROM golang:1.21.6-alpine AS gobuild
## 设置 go 代理 走国内代理
ENV GOPROXY https://goproxy.cn,direct
## 替换 安装镜像
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && apk add --no-cache upx

## workdir 路径 
## 通过 docker exec -it 容器id sh 会直接进入这个目录
WORKDIR /app
## 拷贝 go.mod 文件进行依赖下载 '.' 符号为 workdir
COPY go.mod .
RUN go mod download && go mod verify
## 拷贝 所有文件到工作目录
COPY main.go .
## 构建 
RUN go build -ldflags "-s -w" -o webhook2email
RUN upx -9 webhook2email

## 第二次构建 也就是go运行环境
FROM alpine:3.14

WORKDIR /app
## 将 第一阶段 AS gobuild 的产物拷贝到 二次构建的运行环境进行运行
COPY --from=gobuild /app/webhook2email .
COPY mail.html .
CMD [ "/app/webhook2email" ]