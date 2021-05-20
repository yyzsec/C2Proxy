package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestName(t *testing.T) {
	get, err := http.Get("http://127.0.0.1/")
	if err != nil {
		panic(err)
	}
	defer get.Body.Close()
	//heads = map[string]string{}
	heads := get.Header
	for i := range heads {
		fmt.Println(heads[i])
	}
	content, err := ioutil.ReadAll(get.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(content))
}