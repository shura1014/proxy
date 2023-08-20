package proxy

import (
	"github.com/shura1014/acl"
	"github.com/shura1014/common/container/tree"
	"github.com/shura1014/common/utils/fileutil"
	"github.com/shura1014/httpclient"
	"net/http"
	"time"
)

type Proxy struct {
	client    *httpclient.Client
	proxyTree *tree.Node
	proto     string
	// 监听的地址
	bind      string
	timeout   time.Duration
	locations map[string]*Location

	// 黑白名单 一般来说都是处理黑名单
	// 如果使用了deny all,那么名单是白名单
	White     bool
	denyList  []string
	allowList []string
	denyFile  string

	// 是否允许响应头显示Server
	enableServer bool

	// 是否需要开启健康检查
	enableHealthCheck bool

	acl acl.Acl
}

func New() *Proxy {
	proxy := &Proxy{
		proxyTree: &tree.Node{
			Name:     "/",
			Children: make([]*tree.Node, 0),
		},
		proto:        ConfigHTTP,
		locations:    make(map[string]*Location),
		client:       httpclient.NewClient(),
		enableServer: true,
		acl:          acl.Default(),
	}

	return proxy
}

func (proxy *Proxy) EnableHealthCheck() {
	proxy.enableHealthCheck = true
}

func (proxy *Proxy) Start() {
	if proxy.proto == ConfigHTTP {
		if proxy.enableHealthCheck {
			InitCheck()
			proxy.healthCheck()
		}
		proxy.Run()
	}
}

func (proxy *Proxy) Run() {
	http.Handle("/", proxy)
	err := http.ListenAndServe(proxy.bind, nil)
	if err != nil {
		Fatal(err)
	}
}

func (proxy *Proxy) AddLocation(location ...*Location) {
	for _, l := range location {
		proxy.locations[l.Rule] = l
		proxy.proxyTree.Put(l.Rule)
	}
}

func (proxy *Proxy) InitAclNodeList(fileName string) {

	var lines []string
	fileutil.DirFunc(configDir, func() {
		lines = fileutil.Read(fileName)
	})

	if len(lines) > 0 {
		// 开启
		proxy.acl.Enable()
		Info("----------------------- ip acl -------------------------------")
		Info("acl path %s", fileName)
		for _, line := range lines {
			proxy.acl.ParseAclNode(line)
			Info(line)
		}
		Info("----------------------------------------------------------------------------")
	}
}
