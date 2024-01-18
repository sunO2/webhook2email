## 使用和 go.mod 文件中相通的 go 版本 编译go文件
## 通过使用 AS 来 进行多阶段构建
FROM golang:1.21.6-alpine AS gobuild
## 设置 go 代理 走国内代理
ENV GOPROXY https://goproxy.cn,direct
## workdir 路径 
## 通过 docker exec -it 容器id sh 会直接进入这个目录
WORKDIR /app
## 拷贝 go.mod 文件进行依赖下载 '.' 符号为 workdir
COPY go.mod .
RUN go mod download && go mod verify
## 拷贝 所有文件到工作目录
COPY main.go .
## 构建 
RUN go build -o webhook2email

## 第二次构建 也就是go运行环境
FROM alpine:3.14

WORKDIR /
## 将 第一阶段 AS gobuild 的产物拷贝到 二次构建的运行环境进行运行
COPY --from=gobuild /app/webhook2email ./
CMD [ "webhook2email" ]