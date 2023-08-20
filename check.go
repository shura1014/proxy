package proxy

import (
	"bufio"
	"context"
	"github.com/shura1014/common/goerr"
	"github.com/shura1014/common/utils/stringutil"
	"github.com/shura1014/httpclient"
	"github.com/shura1014/task"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var scheduler *task.Scheduler

func InitCheck() {
	scheduler = task.New()
	scheduler.Start()
}

type Check struct {
	enable bool
	// 间隔时间，毫秒
	interval time.Duration
	// 连续多少次成功算探测成功
	success int
	// 连续多少次失败就算失败
	fail int
	// 超时时间，毫秒
	timeout time.Duration

	send string

	mode string
}

func (proxy *Proxy) healthCheck() {
	for _, location := range proxy.locations {
		check := location.Check
		servers := location.Servers
		jobName := location.Rule
		for i, server := range servers {
			// 如果不需要健康检查，那么所有的ip都是有效
			if !proxy.enableHealthCheck {
				server.valid = true
				continue
			}

			// 需要健康检查，定时检查
			// 解决匿名函数引用同一变量问题
			server := server
			scheduler.ScheduleSingletonNowTask(context.TODO(), check.interval*time.Millisecond, func(ctx context.Context) {
				if check.mode == ConfigHTTP {
					req, _ := http.ReadRequest(bufio.NewReader(strings.NewReader(check.send)))
					if req == nil {
						Error("The Check is valid %s", check.send)
					}
					req.URL, _ = url.Parse(proxy.proto + "://" + server.Address + req.RequestURI)
					req.RequestURI = ""

					timeout, cancelFunc := context.WithTimeout(req.Context(), 2000*time.Millisecond)
					defer cancelFunc()
					do, err := proxy.client.Do(&httpclient.Request{Request: req.WithContext(timeout)})
					if err != nil {
						Error(err.(*goerr.BizError).DetailMsg())
					}
					if do != nil && err == nil && do.StatusCode() == http.StatusOK {
						if !server.valid {
							Info("%s connect success", server.Address)
						}
						server.Success()
					} else {
						server.Fail()
					}
				} else if check.mode == ConfigTCP {
					conn, err := net.DialTimeout(ConfigTCP, server.Address, check.timeout*time.Millisecond)
					if conn != nil {
						if !server.valid {
							Info("%s connect success", server.Address)
						}
						server.Success()
						_ = conn.Close()
						return
					}
					if err != nil {
						Error(err.Error())
					}
					server.Fail()
				}
			}, jobName+"-"+stringutil.ToString(i))
		}
	}

}
