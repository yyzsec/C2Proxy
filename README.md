# 使用golang云函数转发http流量
最近在做红队相关，所以设想了一下隐藏红队C2服务器的骚操作--使用云函数转发http/https流量来隐藏cobaltstrike服务器的http/https beacon。
## 阅读golang云函数基本写法
[看腾讯云的文档](https://cloud.tencent.com/document/product/583/18032)
golang的函数形态一般如下：
```go
package main

import (
    "context"
    "fmt"
    "github.com/tencentyun/scf-go-lib/cloudfunction"
)

type DefineEvent struct {
    // test event define
    Key1 string `json:"key1"`
    Key2 string `json:"key2"`
}

func hello(ctx context.Context, event DefineEvent) (string, error) {
    fmt.Println("key1:", event.Key1)
    fmt.Println("key2:", event.Key2)
    return fmt.Sprintf("Hello %s!", event.Key1), nil
}

func main() {
    // Make the handler available for Remote Procedure Call by Cloud Function
    cloudfunction.Start(hello)
}
```
代码开发时，请注意以下几点：

* 需要使用 package main 包含 main 函数。
* 引用 github.com/tencentyun/scf-go-lib/cloudfunction 库，在编译打包之前，执行 go get github.com/tencentyun/scf-go-lib/cloudfunction。
* 入口函数入参可选0 - 2参数，如包含参数，需 context 在前，event 在后，入参组合有 （），（event），（context），（context，event），具体说明请参见 入参。
* 入口函数返回值可选0 - 2参数，如包含参数，需返回内容在前，error 错误信息在后，返回值组合有 （），（ret），（error），（ret，error），具体说明请参见 返回值。
* 入参 event 和返回值 ret，均需要能够兼容 encoding/json 标准库，可以进行 Marshal、Unmarshal。
* 在 main 函数中使用包内的 Start 函数启动入口函数

## Hello world

上传go源码的时候必须上传zip包

```powershell
# powershell
$env:GOOS=linux
$env:GOARCH=amd64
go build -o main main.go
```

然后测试一下，正确返回结果。

## API网关触发云函数

因为我们需要转发http请求，那么我们的触发条件需要改成API网关触发

参阅腾讯云给的[github-Demo](https://github.com/tencentyun/scf-go-lib/blob/master/events/apigw.go)

### /events/events.go

```go
  
package events

import (
	"encoding/json"
	"fmt"
)

// APIGatewayRequestContext represents a request context
type APIGatewayRequestContext struct {
	ServiceID string `json:"serviceId"`
	RequestID string `json:"requestId"`
	Method    string `json:"httpMethod"`
	Path      string `json:"path"`
	SourceIP  string `json:"sourceIp"`
	Stage     string `json:"stage"`
	Identity  struct {
		SecretID *string `json:"secretId"`
	} `json:"identity"`
}

// APIGatewayRequest represents an API gateway request
type APIGatewayRequest struct {
	Headers     map[string]string        `json:"headers"`
	Method      string                   `json:"httpMethod"`
	Path        string                   `json:"path"`
	QueryString APIGatewayQueryString    `json:"queryString"`
	Body        string                   `json:"body"`
	Context     APIGatewayRequestContext `json:"requestContext"`

	// the following fields are ignored
	// HeaderParameters      interface{} `json:"headerParameters"`
	// PathParameters        interface{} `json:"pathParameters"`
	// QueryStringParameters interface{} `json:"queryStringParameters"`
}

// APIGatewayResponse represents an API gateway response
type APIGatewayResponse struct {
	IsBase64Encoded bool              `json:"isBase64Encoded"`
	StatusCode      int               `json:"statusCode"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
}

// APIGatewayQueryString represents query string of an API gateway request
type APIGatewayQueryString map[string][]string

// UnmarshalJSON implements the json.Unmarshaller interface,
// it handles the query string properly
func (qs *APIGatewayQueryString) UnmarshalJSON(data []byte) error {
	m := make(map[string]interface{})
	err := json.Unmarshal(data, &m)
	if err != nil {
		return err
	}
	r := make(APIGatewayQueryString)
	for k, v := range m {
		switch v.(type) {
		case bool:
			r[k] = []string{}
		case string:
			r[k] = []string{v.(string)}
		case []string:
			r[k] = v.([]string)
		case []interface{}:
			vs := v.([]interface{})
			for _, sv := range vs {
				s, ok := sv.(string)
				if !ok {
					return fmt.Errorf("unexpected query string value: %+v, type: %T", v, v)
				}
				r[k] = append(r[k], s)
			}
		default:
			return fmt.Errorf("unexpected query string value: %+v, type: %T", v, v)
		}
	}
	*qs = r
	return nil
}
```

* 实际上根据上面的注意事项写函数即可，注意出口与入口
* 这是定义了给我们的处理http请求request入口与response的出口的相关函数与接口

### main.go

```go
package main

import (
	"context"
	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"github.com/yyz/C2Proxy/events"
)

// Run 执行
func Run(ctx context.Context, event events.APIGatewayRequest) (resp events.APIGatewayResponse, err error) {

	resp = events.APIGatewayResponse{
		IsBase64Encoded: false,
		Headers:         map[string]string{},
		StatusCode: 200,
		Body: "Hello World!",
	}
	return

}

func proxy() () {


}

func main() {
	cloudfunction.Start(Run)
}

```

* 调试的时候可以看一下[状态码信息文档](https://cloud.tencent.com/document/product/583/42611)，中途我好几次443都是因为函数入口没写对导致了UserCodeError
* 可以发现整个http的response都是我们能控制的

## 代理http协议

首先模拟一下简单的请求其他http页面然后返回回来

```go
package main

import (
	"context"
	"fmt"
	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"github.com/yyz/C2Proxy/events"
	"io/ioutil"
	"net/http"
)

// Run 执行
func Run(ctx context.Context, event events.APIGatewayRequest) (resp events.APIGatewayResponse, err error) {
	resp = events.APIGatewayResponse{
		IsBase64Encoded: false,
		Headers: map[string]string{},
		StatusCode: 502,
		Body: "error",
	}
	// x.x.x.x是我的vps地址，在上面起了一个python -m http.server
	get, err := http.Get("http://x.x.x.x:8083/")
	if err != nil {
		resp.Body = err.Error()
		return
	}
	heads := get.Header
	for i := range heads {
		fmt.Println(heads[i])
		resp.Headers[i] = heads[i][0]
	}
	resp.StatusCode = get.StatusCode
	content, err := ioutil.ReadAll(get.Body)
	if err != nil {
		resp.Body = err.Error()
		return
	}
	resp.Body = string(content)
	return

}


func main() {
	cloudfunction.Start(Run)
}

```

> x.x.x.x是我的vps地址，在上面起了一个python -m http.server

可以看到正确返回了代理请求

![httpreq](https://yyz9.cn/images/httpreq.jpg)

需要返回的内容：

* IsBase64Encode: bool
* StatusCode: int
* Headers: map[string]string
* Body: string

需要处理的request

* Headers: map[string]string
* Method: string
* Path: string
* QueryString: map\[string][]string
* Body: string
* Context: APIGatewayRequestContext

```go
type APIGatewayRequestContext struct {
	ServiceID string `json:"serviceId"`
	RequestID string `json:"requestId"`
	Method    string `json:"httpMethod"`
	Path      string `json:"path"`
	SourceIP  string `json:"sourceIp"`
	Stage     string `json:"stage"`
	Identity  struct {
		SecretID *string `json:"secretId"`
	} `json:"identity"`
}
```

## 踩坑

* 在浏览器中的路由如果是以/golangTest结束，然后再点击页面中的链接的时候会把golangTest路由给挤掉，导致访问的不正确
* 解决url的路由问题的话，可以在API网关设置里面把/golangTest给换成/

那我们直接看看一次请求的APIRequest打印出来是什么

```go
	fmt.Println("下面是event：")
	fmt.Println(event)
	fmt.Println("结束event打印")
```

```go
下面是event：

{map[Accept:text/html,application/xml,application/json Accept-Language:en-US,en,cn Host:service-3ei3tii4-251000691.ap-guangzhou.apigateway.myqloud.com User-Agent:User Agent String] POST /flag map[bob:[alice] foo:[bar]] "This is post body" {service-f94sy04v c6af9ac6-7b61-11e6-9a41-93e8deadbeef POST /test/{path} 10.0.2.14 release {0xc00014e2b0}}}

结束event打印
```

那么我们就可以转发任意http请求并返回了

## 最终代码[GET/POST]

实际上http是不面向连接的，每个http单独转发然后返回就行了，我们只需要在之前代码的基础上加一下请求的Path和qureyString就行了，如果是POST的话再处理一下Body

```go
package main

import (
	"context"
	"fmt"
	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"github.com/yyz/C2Proxy/events"
	"io/ioutil"
	"net/http"
	"strings"
)

var HTTPresp *http.Response
// Run 执行
func Run(ctx context.Context, event events.APIGatewayRequest) (resp events.APIGatewayResponse, err error) {
	resp = events.APIGatewayResponse{
		IsBase64Encoded: false,
		Headers: map[string]string{},
		StatusCode: 502,
		Body: "error",
	}
	// 下面这一句replace其实是不需要的
	resqPath := strings.Replace(event.Path, "/golangTest", "", -1)
	// 下面这几行是处理QueryString
	qrString := ""
	lenQRString := len(event.QueryString)
	if lenQRString != 0 {
		qrString = "?"
		for i := range event.QueryString {
			qrString += string(i) + "=" + string(event.QueryString[i][0]) + "&"
		}
		if string(qrString[len(qrString)-1]) == "&" {
			qrString = qrString[:len(qrString)-1]
		}
	}
	// 下面开始判断是GET还是POST请求
	if event.Method == "GET" {
		HTTPresp, err = http.Get("https://www.baidu.com" + resqPath + qrString)
		if err != nil {
			resp.Body = err.Error()
			return
		}
		defer HTTPresp.Body.Close()
	} else if event.Method == "POST" {
		if event.Headers["Content-Type"] == "" {
			event.Headers["Content-Type"] = "application/x-www-form-urlencoded"
		}
		HTTPresp, err = http.Post("https://www.baidu.com" + resqPath + qrString, event.Headers["Content-Type"], strings.NewReader(event.Body))
		if err != nil {
			resp.Body = err.Error()
			return
		}
		defer HTTPresp.Body.Close()
	}


	// 下面是得到了http response 然后把他封装成API特定的返回
	// 需要封装resp.Headers、resp.StatusCode、resp.Body、resp.IsBase64Encoded
	heads := HTTPresp.Header
	for i := range heads {
		fmt.Println(heads[i])
		resp.Headers[i] = heads[i][0]
	}
	resp.StatusCode = HTTPresp.StatusCode
	content, err := ioutil.ReadAll(HTTPresp.Body)
	if err != nil {
		resp.Body = err.Error()
		return
	}
	resp.Body = string(content)
	// 下面是调试用的代码
	fmt.Println("下面是event：")
	fmt.Println(event)
	fmt.Println("结束event打印")
	fmt.Println("下面是HTTPresp URL：")
	fmt.Println("http://www.baidu.com" + resqPath + qrString)
	fmt.Println("结束HTTPresp URL打印")
	return

}


func main() {
	cloudfunction.Start(Run)
}

```

这里我直接代理了https请求发往百度，浏览器打开看一下可以正常加载百度首页，并且搜索功能正常。

至此，一个简单的基于云函数&golang的http/https代理完成了。

![baidu](https://yyz9.cn/images/baidu.jpg)

## 接下来

* 再多看看CS官方文档(正在做.....)
* 修改一下CS源码流量等特征(正在做.....)
* 上线
* 代理C2服务器