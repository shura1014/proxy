package proxy

import (
	"github.com/shura1014/balance"
	"github.com/shura1014/common/utils/fileutil"
	"github.com/shura1014/common/utils/stringutil"
	"github.com/shura1014/common/utils/timeutil"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	defaultFilePath = "./etc/proxy.conf"
)

const (
	ConfigHTTP       = "http"
	ConfigTCP        = "tcp"
	ConfigLocation   = "location"
	ConfigExpression = "expression"
	IF               = "if"
	BIND             = "bind"
	ObjEnd           = "}"
	ObjStart         = "{"
	SPACE            = " "
	// BALANCE ********************location****************
	BALANCE         = "balance"
	HealthCheck     = "health_check"
	HealthCheckSend = "health_check_send"
	SERVER          = "server"
	AddHeader       = "add_header"
	checkReg        = `^\s*([\w]+)=([\w]+)\s+([\w]+)=([\w]+)\s+([\w]+)=([\w]+)\s+([\w]+)=([\w]+)\s+([\w]+)=([\w]+)\s*$`
	// INTERVAL FAIL SUCCESS TIMEOUT Check
	INTERVAL = "interval"
	SUCCESS  = "success"
	FAIL     = "fail"
	TIMEOUT  = "timeout"
	MODE     = "mode"
	ROOT     = "root"
	INDEX    = "index"
	DENY     = "deny"
	ALLOW    = "allow"
	All      = "all"
	Include  = "include"
)

var (
	configDir    string
	baseDir      string
	confFileName string
)

// ParseConfig 开始解析配置文件
// filepath文件路径
func ParseConfig(filepath string) (proxys []*Proxy) {
	if filepath == "" {
		filepath = defaultFilePath
	}
	baseDir = deduceDir(".")

	var fileContent []string
	fileutil.DirFunc(baseDir, func() {
		confFileName = fileutil.FileName(filepath)
		configDir = strings.TrimSuffix(fileutil.RealPath(filepath), confFileName)
		fileContent = fileutil.Read(filepath)
	})

	return ParseContent(fileContent)
}

func ParseContent(lines []string) (proxys []*Proxy) {
	var (
		// 当前处于哪一个快中
		curBlock = ""
		// 当前正在解析的 Proxy
		curProxy      *Proxy
		curExpression *Expression
		// 当前正在解析的 Location
		curLocation *Location
	)
	for _, line := range lines {
		// 去掉注释 约定写在后面的注释只能使用 #
		index := strings.Index(line, "#")
		if index != -1 {
			line = line[0:index]
		}
		// 去除两边空格
		line = strings.Trim(line, SPACE)

		// 如果这一行都是注释，不做处理
		if strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		switch curBlock {
		case "":
			if strings.HasPrefix(line, ConfigHTTP) && strings.HasSuffix(line, ObjStart) {
				// 开始进入http快
				curBlock = ConfigHTTP
				curProxy = New()
				curProxy.proto = ConfigHTTP
				continue
			}

			if strings.HasPrefix(line, ConfigTCP) && strings.HasSuffix(line, ObjStart) {
				// 开始进入tcp快 todo
				curBlock = ConfigTCP
				curProxy.proto = ConfigTCP
				continue
			}
		case "http":
			before, after, _ := strings.Cut(line, " ")
			if before != "" {
				switch before {
				case BIND:
					curProxy.bind = strings.Trim(after, SPACE)
				case TIMEOUT:
					curProxy.timeout = timeutil.Convert(strings.Trim(after, SPACE)) * time.Millisecond
				case DENY:
					// 黑名单
					denyLine := strings.Trim(after, SPACE)
					denyType, deny, _ := strings.Cut(denyLine, " ")
					switch denyType {
					case All:
						curProxy.White = true
					case Include:
						curProxy.denyFile = strings.Trim(deny, SPACE)
					default:
						curProxy.denyList = append(curProxy.denyList, denyType)
					}

				case ALLOW:
					curProxy.allowList = append(curProxy.allowList, strings.Trim(after, SPACE))
				case SERVER:
					enable := strings.Trim(after, SPACE)
					if enable == "off" {
						curProxy.enableServer = false
					}
					if enable == "on" {
						// 默认就是on
						curProxy.enableServer = true
					}
				case ConfigLocation:
					if strings.HasSuffix(after, ObjStart) {
						curBlock = ConfigLocation
						rule := strings.TrimFunc(after, func(r rune) bool {
							if r == ' ' || r == '{' {
								return true
							}
							return false
						})
						if rule != "" {
							curLocation = NewLocation()
							curLocation.Rule = rule
						}
					}
				case ObjEnd:
					if curProxy.timeout > 0 {
						curProxy.client.SetTimeout(curProxy.timeout)
					}

					if curProxy.White {
						// 白名单处理，手动开启Enable
						curProxy.acl.Enable()
						for _, allow := range curProxy.allowList {
							curProxy.acl.ParseAclNode(allow)
						}

					} else {
						if len(curProxy.denyList) > 0 || curProxy.denyFile != "" {
							curProxy.acl.Enable()
							if curProxy.denyFile != "" {
								fileutil.DirFunc(configDir, func() {
									curProxy.InitAclNodeList(curProxy.denyFile)
								})
							}
							for _, deny := range curProxy.denyList {
								curProxy.acl.ParseAclNode(deny)
							}
						}
					}

					proxys = append(proxys, curProxy)
					curProxy = nil
					curBlock = ""
				}
			}
		case "location":
			before, after, _ := strings.Cut(line, " ")
			if before != "" {
				switch before {
				case ROOT:
					// 是一个静态资源服务器
					curLocation.IsStatic = true
					curLocation.Root = strings.Trim(after, SPACE)
				case INDEX:
					// 如果不是静态资源服务器那么不处理
					// 所以约定root应当在index的前面配置
					if !curLocation.IsStatic {
						continue
					}
					curLocation.Index = strings.Trim(after, SPACE)
				case BALANCE:
					b := strings.Trim(after, SPACE)
					// 由于Go的泛型还不够强大造成
					if b == balance.Weight_ {
						curLocation.Balance = balance.NewWeight[*Server]()
					} else {
						curLocation.Balance = balance.GetBalance[*Server](b)
					}
				case HealthCheck:
					reg, err := regexp.Compile(checkReg)
					if err != nil {
						Error("Check 配置错误 请按如下格式配置\n%s %v", "Check interval=2000 success=2 fail=3 timeout=1000 mode=http", err)
					}
					matchs := reg.FindStringSubmatch(after)
					checkMap := make(map[string]string)
					if len(matchs) != 11 {
						Fatal("%s config Check failed", curLocation.Rule)
					}
					for i := 1; i < len(matchs); i += 2 {
						checkMap[matchs[i]] = matchs[i+1]
					}
					curLocation.Check.enable = true
					v, ok := checkMap[SUCCESS]
					if ok {
						s, _ := strconv.Atoi(v)
						curLocation.Check.success = s
					}
					v, ok = checkMap[FAIL]
					if ok {
						s, _ := strconv.Atoi(v)
						curLocation.Check.fail = s
					}

					v, ok = checkMap[INTERVAL]
					if ok {
						curLocation.Check.interval = timeutil.Convert(v)
					}
					v, ok = checkMap[TIMEOUT]
					if ok {
						curLocation.Check.timeout = timeutil.Convert(v)
					}
					v, ok = checkMap[MODE]
					curLocation.Check.mode = v
					// 开启了健康检查
					curProxy.EnableHealthCheck()
				case HealthCheckSend:
					after = strings.ReplaceAll(after, "\\r", "\r")
					after = strings.ReplaceAll(after, "\\n", "\n")
					curLocation.Check.send = strings.TrimFunc(after, func(r rune) bool {
						switch r {
						case ' ':
							return true
						case '"':
							return true
						default:
							return false
						}
					})
				case SERVER:
					// 权重负载处理
					if curLocation.Balance.Name() == balance.Weight_ {
						trim := strings.Trim(after, SPACE)
						weight := 1
						before, after, _ = strings.Cut(trim, " ")
						// weight=1
						if after != "" && len(after) > 7 {
							weight, _ = strconv.Atoi(strings.Trim(after[7:], SPACE))
						}
						s := &Server{Address: strings.Trim(before, SPACE), weight: weight, check: curLocation.Check}
						curLocation.Servers = append(curLocation.Servers, s)
					} else {
						// 如果不是weight，那么就算配置了weight也不起作用
						// 如果多余配置了weight，需要去掉
						trim := strings.Trim(after, SPACE)
						before, after, _ = strings.Cut(trim, " ")
						s := &Server{Address: before, check: curLocation.Check}
						curLocation.Servers = append(curLocation.Servers, s)
					}
				case AddHeader:
					trim := strings.Trim(after, SPACE)
					before, after, _ = strings.Cut(trim, " ")
					after = strings.Trim(after, SPACE)
					if before != "" && after != "" {
						before = strings.ReplaceAll(before, "'", "")
						after = strings.ReplaceAll(after, "'", "")
						curLocation.headers[before] = after
					}
				case ValidReferer:
					curLocation.referer = NewReferer()
					args := strings.TrimFunc(after, func(r rune) bool {
						if r == ' ' || r == '{' {
							return true
						}
						return false
					})
					split := strings.Split(args, " ")
					for _, arg := range split {

						if arg == None {
							curLocation.referer.validType = None
						}

						if strings.HasPrefix(arg, Blocked) {
							arg = arg[8:]
							curLocation.referer.domain = strings.Split(arg, ",")
						}
					}
				case IF:
					if strings.HasSuffix(after, ObjStart) {
						curBlock = ConfigExpression
						// if $request_method = 'OPTIONS' {
						expression := strings.TrimFunc(after, func(r rune) bool {
							if r == ' ' || r == '{' {
								return true
							}
							return false
						})
						split := strings.Split(expression, " == ")
						var (
							key    string
							expect string
						)
						if len(split) > 2 {
							panic("Wrong expression " + line)
						}
						if len(split) == 1 {
							key = split[0]
						}

						if len(split) == 2 {
							key = split[0]
							expect = split[1]
						}
						key = key[1:]
						if !stringutil.IsArray(SupportKey, key) {
							panic("Unsupported expression key " + key)
						}
						curExpression = NewExpression()
						curExpression.key = key
						curExpression.expect = strings.ReplaceAll(expect, "'", "")
					}

				case ObjEnd:
					// location解析结束，添加到相应的proxy中
					if curLocation.Balance != nil {
						curLocation.Balance.InitNodes(curLocation.Servers...)
					}
					curProxy.AddLocation(curLocation)
					// 滞空
					curLocation = nil
					curBlock = curProxy.proto
				default:

				}
			}
		case ConfigExpression:
			before, after, _ := strings.Cut(line, " ")
			switch before {
			case Return:
				code, _ := strconv.Atoi(strings.Trim(after, SPACE))
				curExpression.returnCode = code
			case AddHeader:
				trim := strings.Trim(after, SPACE)
				before, after, _ = strings.Cut(trim, " ")
				after = strings.Trim(after, SPACE)
				if before != "" && after != "" {
					before = strings.ReplaceAll(before, "'", "")
					after = strings.ReplaceAll(after, "'", "")
					curExpression.headers[before] = after
				}
			case ObjEnd:
				curLocation.expression = append(curLocation.expression, curExpression)
				// 滞空
				curExpression = nil
				curBlock = ConfigLocation
			default:

			}
		default:
			panic("unknown")
		}
	}
	return
}

func deduceDir(dir string) string {
	if dir == "" {
		return ""
	}
	if strings.HasPrefix(dir, "/") {
		// 绝对路径
		return dir
	}

	// 	相对路径
	var realPath string
	_, file, _, _ := runtime.Caller(2)
	fileutil.DirFunc(fileutil.Dir(file), func() {
		realPath = fileutil.RealPath(dir)
	})
	return realPath
}
