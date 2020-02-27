package main

import (
	"/http_util"
	"fmt"
	"net/http"
	"time"
)

func main() {
	client := http_util.NewClient("http://hmzp.ex4.ink:10080/put_offer_apply/", http.MethodPost)
	client.Timeout = time.Millisecond * 500  // timeout
	client.ReTry = 5  // 重试次数
	client.Body = http_util.Body{  // body params
		ContentType: http_util.FormData,
		Data:        http_util.Data{"job_id": "2"},
	}
	res, err := client.Send()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(res.Text())
		fmt.Println(res.Cookies)
		fmt.Println(res.Headers)
	}
}

