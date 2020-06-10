package polarion

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Client struct {
	url   string
	debug bool
}

func NewClient(url string, debug bool) *Client {
	return &Client{url: url, debug: debug}
}

func (c *Client) request(service service, request, response interface{}) error {

	url := fmt.Sprintf("%s/%s", c.url, service)

	p, err := xml.Marshal(request)
	if err != nil {
		return err
	}

	if c.debug {
		fmt.Println("request:", string(p))
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(p))
	if err != nil {
		return err
	}

	req.Header.Set("Content-type", "text/xml")
	req.Header.Set("SOAPAction", "")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	d, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	if c.debug {
		fmt.Println("response:", string(d))
	}
	return xml.Unmarshal(d, response)
}

func (c *Client) LogIn(username, password string) (*LogInEnvelopResponse, error) {

	request := NewLogInRequest(LogInRequestBodyLogIn{
		UserName: username,
		Password: password,
	})

	respose := &LogInEnvelopResponse{}

	err := c.request(sessionService, request, respose)
	if err != nil {
		return nil, err
	}

	if respose.Body.Fault.Faultcode != "" {
		return nil, fmt.Errorf("logIn request failed with error: %s", respose.Body.Fault.Faultstring)
	}

	return respose, nil
}
