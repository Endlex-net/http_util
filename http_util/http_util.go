package http_util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// 使用 Demo
//func UseDemo() {
//	client := NewClient("http://www.baidu.com", http.MethodPost)
//	client.QueryParams = map[string]string{"test": "test"}
//	client.Body = Body{
//		ContentType: Urlencoded,
//		Data:        map[string]string{"test": "test"},
//	}
//	client.ReTry = 5                        // NewClient 默认 0 次
//	client.Timeout = 500 * time.Millisecond // NewClient 默认 10秒
//	client.ReCheck = func(response *Response) bool { // 访问成功后 重试判断函数
//		return len(response.Body) < 100
//	}
//	req, err := client.Send()
//	if err != nil {
//		fmt.Println(err.Error())
//	}
//	fmt.Println(req.Text())
//}

type Data map[string]string

func (d *Data) Copy() Data {
	newData := Data{}
	for k, v := range *d {
		newData[k] = v
	}
	return newData
}

const (
	FormData   = "multipart/form-data"
	Urlencoded = "application/x-www-form-urlencoded"
)

type Body struct {
	ContentType string
	Data        Data
	StrData     string
}

type Client struct {
	Url         string // 请求链接
	Method      string
	QueryParams Data
	Headers     Data
	Body        Body
	Response    *Response            // 返回值
	Timeout     time.Duration        // 超时时间
	ReTry       uint                 // 重试次数
	ReCheck     func(*Response) bool // 重试判断函数(false 需要重试)
}

func DefaultReCheck(res *Response) bool {
	return true
}

// 创建一个新链接
func NewClient(url string, method string) *Client {
	return &Client{
		Url:         url,
		Method:      method,
		QueryParams: Data{},
		Headers:     Data{},
		Body:        Body{},
		Timeout:     10 * time.Second, // 默认 10秒
		ReTry:       0,                // 默认 0次
		ReCheck:     DefaultReCheck,
	}
}

func (c *Client) Copy() Client {
	return Client{
		Url:         c.Url,
		Method:      c.Method,
		QueryParams: c.QueryParams.Copy(),
		Headers:     c.Headers.Copy(),
		Body: Body{
			ContentType: c.Body.ContentType,
			Data:        c.Body.Data.Copy(),
			StrData:     c.Body.StrData,
		},
		Timeout: c.Timeout,
		ReTry:   c.ReTry,
		ReCheck: c.ReCheck,
	}
}

// 请求
func (c *Client) Send() (*Response, error) {
	res := new(Response)
	var err error

	// 重试
	for i := 0; i < int(c.ReTry+1); i++ {
		client := c.Copy()
		res, err = client.send()
		if err == nil && client.ReCheck(res) {
			break
		} else {
			// 指数避让
			time.Sleep(binaryExponentialBackOff(i))
		}
	}
	return res, err
}

// 指数避让
func binaryExponentialBackOff(i int) time.Duration {
	k := math.Pow(2, float64(i))
	baseTime := 50 * time.Millisecond
	sleepTime := time.Duration(k) * baseTime
	fmt.Println("try:", i, "sleep:", sleepTime)
	return sleepTime
}

// 单次请求
func (c *Client) send() (*Response, error) {
	url := c.getQueryUrl()
	body := c.serializeBody()
	req, err := http.NewRequest(c.Method, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Set Headers
	for key, value := range c.Headers {
		req.Header.Add(key, value)
	}
	req.Header.Set("Content-Type", c.Body.ContentType)

	client := &http.Client{}
	client.Timeout = c.Timeout
	httpRes, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	res, err := NewResponse(httpRes)
	if r := recover(); r != nil {
		return nil, r.(error)
	}
	return res, err
}

// 获得置入QueryParams的url
func (c *Client) getQueryUrl() string {
	if len(c.QueryParams) > 0 {
		return c.Url + "?" + linkData(c.QueryParams)
	}
	return c.Url
}

// &连接Data
func linkData(data Data) string {
	var items []string
	for key, value := range data {
		item := key + "=" + value
		items = append(items, item)
	}
	return strings.Join(items, "&")
}

// serializeBody
func (c *Client) serializeBody() string {
	contentType := ""
	body := ""
	switch c.Body.ContentType {
	case Urlencoded:
		contentType = Urlencoded
		body = linkData(c.Body.Data)
	case FormData:
		b := &bytes.Buffer{}
		writer := multipart.NewWriter(b)
		for key, value := range c.Body.Data {
			_ = writer.WriteField(key, value)
		}
		_ = writer.Close()
		contentType = writer.FormDataContentType()
		body = b.String()
	default:
		contentType = ""
		body = c.Body.StrData
	}
	c.Body.ContentType = contentType
	return body
}

// Response
type Response struct {
	Body    []byte
	Cookies Data
	Headers Data
}

func NewResponse(httpRes *http.Response) (*Response, error) {
	// body
	body, err := ioutil.ReadAll(httpRes.Body)
	if err != nil {
		return nil, err
	}

	// cookies
	cookies := Data{}
	for _, cookie := range httpRes.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}

	// headers
	headers := Data{}
	for key, values := range httpRes.Header {
		headers[key] = strings.Join(values, ";")
	}

	return &Response{
		Body:    body,
		Cookies: cookies,
		Headers: headers,
	}, nil
}

// Json
func (r *Response) Json(v interface{}) error {
	err := json.Unmarshal(r.Body, v)
	return err
}

// Text
func (r *Response) Text() string {
	if r.Body == nil {
		return ""
	}
	return string(r.Body)
}
