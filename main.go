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
// Run 是核心处理函数
func Run(ctx context.Context, event events.APIGatewayRequest) (resp events.APIGatewayResponse, err error) {
	// 初始化返回API结构体
	resp = events.APIGatewayResponse{
		IsBase64Encoded: false,
		Headers: map[string]string{},
		StatusCode: 502,
		Body: "error",
	}
	// 下面这几行是处理QueryString
	qrString := ""
	lenQRString := len(event.QueryString)
	if lenQRString != 0 {
		qrString = "?"
		for i := range event.QueryString {
			qrString += i + "=" + event.QueryString[i][0] + "&"
		}
		if string(qrString[len(qrString)-1]) == "&" {
			qrString = qrString[:len(qrString)-1]
		}
	}
	reqsUrl := "https://www.baidu.com" + event.Path + qrString
	// 下面开始判断是GET还是POST请求
	if event.Method == "GET" {
		HTTPresp, err = http.Get(reqsUrl)

		if err != nil {
			resp.Body = err.Error()
			return
		}
		defer HTTPresp.Body.Close()
	} else if event.Method == "POST" {
		if event.Headers["Content-Type"] == "" {
			event.Headers["Content-Type"] = "application/x-www-form-urlencoded"
		}
		HTTPresp, err = http.Post(reqsUrl, event.Headers["Content-Type"], strings.NewReader(event.Body))

		if err != nil {
			resp.Body = err.Error()
			return
		}
		defer HTTPresp.Body.Close()
	}

	// 下面是得到了http response 然后把他封装成API特定的返回的过程
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

	return

}


func main() {
	// 云函数的调用起点
	cloudfunction.Start(Run)
}
