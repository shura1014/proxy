package proxy

import (
	"context"
	"errors"
	"fmt"
	"github.com/shura1014/common/goerr"
	"github.com/shura1014/common/utils/fileutil"
	"github.com/shura1014/common/utils/stringutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-Ip"
)

func (proxy *Proxy) handlerStatic(resp http.ResponseWriter, req *http.Request, location *Location) {
	root := location.Root
	fs := http.Dir(fileutil.Join(baseDir, root))
	fileServer := http.FileServer(fs)
	// 检查一下文件是否存在
	path := req.URL.Path
	// 说明是访问的首页
	if strings.HasPrefix(location.Rule, path) {
		path = fileutil.Join(path, location.Index)
		req.URL.Path = path
	}
	f, err := fs.Open(path)
	if err != nil {
		resp.WriteHeader(http.StatusNotFound)
		return
	}
	_ = f.Close()
	fileServer.ServeHTTP(resp, req)
}

func (proxy *Proxy) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	// 异常捕获，防止程序中断
	defer func() {
		if err := recover(); err != nil {
			Error(err)
		}
	}()
	path := req.URL.Path

	// ip 黑白名单检查 start-----------------------------------------------
	ip := ClientIP(req)
	check := proxy.acl.AclCheck(ip)
	// 检查到了，但是列表全是黑名单处理，那么 return false
	if proxy.acl.IsEnable() && check && !proxy.White {
		_ = String(resp, http.StatusUnauthorized, "Deny illegal access from %s", ip)
		return
	}

	// 没有检查到，但是列表全是白名单处理，那么 return false
	if proxy.acl.IsEnable() && !check && proxy.White {
		_ = String(resp, http.StatusUnauthorized, "Deny illegal access from %s", ip)
		return
	}
	// ip 黑白名单检查 end-----------------------------------------------

	treeNode := proxy.proxyTree.Get(path, nil)

	if treeNode != nil && treeNode.IsEnd {
		// 路由匹配
		location := proxy.locations[treeNode.RouterName]

		if len(location.expression) > 0 {
			for _, expression := range location.expression {
				key := expression.key
				switch key {
				case RequestMethod:
					method := req.Method
					if method == expression.expect && expression.returnCode != 0 {
						ReturnNow(resp, expression.returnCode)
						return
					}
				case RequestPath:
					if path == expression.expect && expression.returnCode != 0 {
						ReturnNow(resp, expression.returnCode)
						return
					}
				case InvalidReferer:
					// 防盗链支持
					if location.referer != nil {
						r := location.referer
						referer := req.Referer()
						if referer == "" && r.validType == None {
							ReturnNow(resp, expression.returnCode)
							return
						}
						if referer != "" {
							if !stringutil.IsArray(r.domain, referer) {
								ReturnNow(resp, expression.returnCode)
								return
							}
						}
					}
				}
			}
		}

		if location.IsStatic {
			// 静态资源服务器
			proxy.handlerStatic(resp, req, location)
			return
		}

		// 代理服务器，转发
		instance, err := location.Balance.Instance()
		if err != nil {
			Error(err.(*goerr.BizError).DetailMsg())
			_ = String(resp, http.StatusServiceUnavailable, "服务不可用")
			return
		}

		// 超时设置
		if proxy.timeout > 0 {
			timeout, cancelFunc := context.WithTimeout(req.Context(), proxy.timeout)
			defer cancelFunc()
			req = req.WithContext(timeout)
		}
		target, _ := url.Parse(fmt.Sprintf(proxy.proto + "://" + instance.Address + path))
		director := func(req *http.Request) {
			req.Host = target.Host
			req.URL.Host = target.Host
			req.URL.Path = target.Path
			req.URL.Scheme = target.Scheme
			if _, ok := req.Header["User-Agent"]; !ok {
				// 禁用
				req.Header.Set("User-Agent", "")
			}
		}

		response := func(response *http.Response) error {
			if proxy.enableServer {
				response.Header.Set("Server", "Gateway/"+VERSION)
			}

			for k, v := range location.headers {
				response.Header.Set(k, v)
			}

			return nil
		}

		handler := func(writer http.ResponseWriter, request *http.Request, err error) {
			if errors.Is(err, context.DeadlineExceeded) {
				if err.(interface {
					Timeout() bool
				}).Timeout() {
					_ = String(resp, http.StatusBadGateway, "request time out")
					return
				}
			}
			_ = String(resp, http.StatusBadGateway, err.Error())
			return
		}
		httpProxy := httputil.ReverseProxy{Director: director, ModifyResponse: response, ErrorHandler: handler}
		httpProxy.ServeHTTP(resp, req)
		return
	}

	_ = String(resp, http.StatusNotFound, "404 not found ")
	return

}

func String(writer http.ResponseWriter, code int, format string, msg ...any) error {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if code != 0 {
		writer.WriteHeader(code)
	}
	if len(msg) > 0 {
		_, err := fmt.Fprintf(writer, format, msg...)
		return err
	}

	data := stringutil.StringToBytes(format)
	_, err := writer.Write(data)
	return err
}

func ReturnNow(writer http.ResponseWriter, code int) {
	if code != 0 {
		writer.WriteHeader(code)
	}
	writer.(http.Flusher).Flush()
}

func ClientIP(req *http.Request) string {
	var clientIP string

	// 检查该ip是不是本地ip 或者说它是一个代理ip，因此需要找出原ip
	proxyIps := req.Header.Get(XForwardedFor)
	if proxyIps != "" {
		// XForwardedFor可能经过多重代理，取第一个就行
		i := strings.IndexAny(proxyIps, ",")
		if i > 0 {
			clientIP = strings.TrimSpace(proxyIps[:i])
		}
		clientIP = strings.TrimPrefix(clientIP, "[")
		clientIP = strings.TrimSuffix(clientIP, "]")
		return clientIP
	}

	clientIP = req.Header.Get(XRealIP)
	if clientIP == "" {
		clientIP, _, _ = net.SplitHostPort(strings.TrimSpace(req.RemoteAddr))
	}

	clientIP = strings.TrimPrefix(clientIP, "[")
	clientIP = strings.TrimSuffix(clientIP, "]")
	return clientIP
}
