package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"github.com/yyz/C2Proxy/events"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)
// Run 是核心处理函数
func Run(ctx context.Context, event events.APIGatewayRequest) (resp events.APIGatewayResponse, err error) {
	// 初始化返回API结构体
	resp = events.APIGatewayResponse{
		IsBase64Encoded: true,
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
	reqsUrl := "http://test.cn:80" + event.Path + qrString
	fmt.Println("===========================")
	fmt.Println(event.Method)
	fmt.Println(event.Body)

	request, err := http.NewRequest(event.Method, reqsUrl, strings.NewReader(event.Body))
	if err != nil {
		log.Println(err)
	}

	for key, value := range event.Headers {
		if key == "host" {
			continue
		}
		request.Header[key] = []string{value}
	}

	request.Header["User-Agent"] = []string{event.Headers["user-agent"]}
	request.Header["X-Forwarded-For"] = []string{event.Context.SourceIP}

	fmt.Println("===========================")
	fmt.Println(request.Header)
	fmt.Println("===========================")


	client := &http.Client{}
	HttpResp, err := client.Do(request)
	if err != nil {
		log.Println(err)
	}
	// 下面是得到了http response 然后把他封装成API特定的返回的过程
	// 需要封装resp.Headers、resp.StatusCode、resp.Body、resp.IsBase64Encoded
	heads := HttpResp.Header
	for i := range heads {
		//fmt.Println(heads[i])
		resp.Headers[i] = heads[i][0]
	}
	resp.StatusCode = HttpResp.StatusCode
	content, err := ioutil.ReadAll(HttpResp.Body)
	if err != nil {
		resp.Body = err.Error()
		return
	}
	resp.Body = base64Encode(content)

	return

}
func base64Encode(raw []byte) (b64 string) {
	b64 = base64.StdEncoding.EncodeToString(raw)
	return
}

func main() {
	// 云函数的调用起点
	cloudfunction.Start(Run)
}
