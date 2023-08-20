# proxy

轻量级的代理与静态资源服务器

# 快速使用

详细请查看 proxy/quickstart

## 创建配置

etc/proxy.conf

```text
# 支持 # 与 ;注释
# # 可以写在任意地方 ; 只能是写在一行的开头
http {
    bind :9000
    ; 超时
    timeout 3000
    ; 黑白名单 一般要买只有deny 要么 allow 与 deny all并存 而单独设置 allow 或者 allow deny没有实际意义
#     deny 127.0.0.0/8 | deny all #如果设置 deny all只有 allow 的名单可以访问
#     deny include ./acl.conf #通过指定的文件配置黑名单
#     allow 127.0.0.1
    deny ::1
    ; 是否需要隐藏Server Server: Gateway/1.0.0
    server off
    location /static/** {
        ; /html/static/**  /html/static 默认访问 index.html
        root html
        index hello.html
        ; 防盗链支持
;         valid_referer none blocked=shura.com;
        if $invalid_referer {
            return 403
        }
    }
    location /api/** {
        balance weight
        ; 2秒发送一次探测 连续两次成功服务可用，连续三次失败服务不可用 探测的超时时间为一秒钟 mode http and tcp
        health_check interval=2000 success=2 fail=3 timeout=1000 mode=http
        health_check_send "HEAD /api/probe HTTP/1.1\r\n\r\n"
        server 127.0.0.1:8888 weight=2
        server 127.0.0.1:8889 weight=3
        ; location级别的timeout
        timeout 3000
        ; 添加响应头
        add_header 'gary' 'true'
        ; 表达式支持
        if $request_method == 'PUT' {
            return 403
        }
    }
}
```

## 启动

可指定配置文件

./proxy -config ./etc/proxy.conf

```text
./proxy 
[task] 2023-08-20 20:52:46 shura1014/task@v1.0.0/scheduler.go:104 INFO scheduler start... 
[task] 2023-08-20 20:52:46 shura1014/task@v1.0.0/registry.go:147 INFO Add task /api/**-0 
[task] 2023-08-20 20:52:46 shura1014/task@v1.0.0/registry.go:147 INFO Add task /api/**-1 
[proxy] 2023-08-20 20:52:48 shura/proxy/check.go:72 INFO 127.0.0.1:8889 connect success 
[proxy] 2023-08-20 20:52:48 shura/proxy/check.go:72 INFO 127.0.0.1:8888 connect success 
[proxy] 2023-08-20 20:52:50 shura/proxy/check.go:72 INFO 127.0.0.1:8889 connect success 
[proxy] 2023-08-20 20:52:50 shura/proxy/check.go:72 INFO 127.0.0.1:8888 connect success
```

## 静态资源访问

```text
curl 127.0.0.1:9000/static/static.txt
测试下载

curl 127.0.0.1:9000/static/          
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Static 测试</title>
</head>
<body>
<h1>hello .</h1>
</body>
</html>%
```

## 代理转发

```text
 curl -v 127.0.0.1:9000/api/remoteIP
*   Trying 127.0.0.1:9000...
* Connected to 127.0.0.1 (127.0.0.1) port 9000 (#0)
> GET /api/remoteIP HTTP/1.1
> Host: 127.0.0.1:9000
> User-Agent: curl/7.79.1
> Accept: */*
> 
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Content-Type: text/plain; charset=utf-8
< Date: Sun, 20 Aug 2023 13:03:02 GMT
< Gary: true
< Transfer-Encoding: chunked
< 
* Connection #0 to host 127.0.0.1 left intact
{"code":200,"msg":"OK","data":"127.0.0.1"}
```

## 黑名单

curl \[::1]:9000/api/remoteIP

Deny illegal access from ::1

## 健康检查测试

```text
先停一个实例再启动
./proxy 
[task] 2023-08-20 20:52:46 shura1014/task@v1.0.0/scheduler.go:104 INFO scheduler start... 
[task] 2023-08-20 20:52:46 shura1014/task@v1.0.0/registry.go:147 INFO Add task /api/**-0 
[task] 2023-08-20 20:52:46 shura1014/task@v1.0.0/registry.go:147 INFO Add task /api/**-1 
[proxy] 2023-08-20 20:52:48 shura/proxy/check.go:72 INFO 127.0.0.1:8889 connect success 
[proxy] 2023-08-20 20:52:48 shura/proxy/check.go:72 INFO 127.0.0.1:8888 connect success 
[proxy] 2023-08-20 20:52:50 shura/proxy/check.go:72 INFO 127.0.0.1:8889 connect success 
[proxy] 2023-08-20 20:52:50 shura/proxy/check.go:72 INFO 127.0.0.1:8888 connect success 
[proxy] 2023-08-20 20:55:02 shura/proxy/check.go:68 ERROR  Error Cause by: 
         Head "http://127.0.0.1:8889/api/probe": dial tcp 127.0.0.1:8889: connect: connection refused
        pkg/mod/github.com/shura1014/httpclient@v0.0.0-20230819033928-25eb909c3d6c/do.go:89
        shura/proxy/check.go:66
        pkg/mod/github.com/shura1014/task@v1.0.0/task.go:151 
[proxy] 2023-08-20 20:55:04 shura/proxy/check.go:68 ERROR  Error Cause by: 
         Head "http://127.0.0.1:8889/api/probe": dial tcp 127.0.0.1:8889: connect: connection refused
        pkg/mod/github.com/shura1014/httpclient@v0.0.0-20230819033928-25eb909c3d6c/do.go:89
        shura/proxy/check.go:66
        pkg/mod/github.com/shura1014/task@v1.0.0/task.go:151 
[proxy] 2023-08-20 20:55:06 shura/proxy/check.go:68 ERROR  Error Cause by: 
         Head "http://127.0.0.1:8889/api/probe": dial tcp 127.0.0.1:8889: connect: connection refused
        pkg/mod/github.com/shura1014/httpclient@v0.0.0-20230819033928-25eb909c3d6c/do.go:89
        shura/proxy/check.go:66
        pkg/mod/github.com/shura1014/task@v1.0.0/task.go:151 
[proxy] 2023-08-20 20:55:08 shura/proxy/check.go:72 INFO 127.0.0.1:8889 connect success 
[proxy] 2023-08-20 20:55:10 shura/proxy/check.go:72 INFO 127.0.0.1:8889 connect success
```

# 配置

## 端口绑定

bind :9000

## 代理超时设置

timeout 3000

## 防盗链配置

```text
valid_referer none blocked=shura.com;
if $invalid_referer {
    return 403
}
```

## 黑白名单

```text
    ; 黑白名单 一般要买只有deny 要么 allow 与 deny all并存 而单独设置 allow 或者 allow deny没有实际意义
;     deny 127.0.0.0/8 | deny all #如果设置 deny all只有 allow 的名单可以访问
;     deny include ./acl.conf #通过指定的文件配置黑名单
;     allow 127.0.0.1
    deny ::1
```

## server

```text
; 是否需要隐藏Server Server: Gateway/1.0.0
server off
```

## 健康检查

支持tcp http

interval=2000 两秒探测一次

success=2 成功两次表示可用

fail=3 失败三次服务不可用

timeout=1000 健康检查超时时间

mode=http http方式

```text
health_check interval=2000 success=2 fail=3 timeout=1000 mode=http
health_check_send "HEAD /api/probe HTTP/1.1\r\n\r\n"
```

两个实例全部停掉

```text
curl 127.0.0.1:9000/api/remoteIP 
服务不可用
```

## 负载均衡策略

支持 roundrobin random weight

例如

balance weight

server 127.0.0.1:8888 weight=2

server 127.0.0.1:8889 weight=3