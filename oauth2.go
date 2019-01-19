package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/MagicalTux/goro/core/util"
)

type OAuth2 struct {
	http.Client
	http.Transport

	token        string
	refreshToken string
	clientId     string
	refresh      time.Time
	endpoint     string
	refreshLock  sync.Mutex
}

type oauth2tokInfo struct {
	Token        string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"` // bearer
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
}

func NewOAuth2(endpoint, clientId, redirectUri, code string) (*OAuth2, error) {
	// first, let's do something about this code
	log.Printf("grabbing token for code client_id=%s code=%s", clientId, code)
	resp, err := http.PostForm(endpoint, url.Values{"grant_type": {"authorization_code"}, "client_id": {clientId}, "redirect_uri": {redirectUri}, "code": {code}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	res := &OAuth2{
		clientId: clientId,
		endpoint: endpoint,
	}
	res.Client.Transport = res

	return res, res.storeToken(body)
}

func (o *OAuth2) storeToken(token []byte) error {
	var data oauth2tokInfo
	err := json.Unmarshal(token, &data)
	if err != nil {
		return err
	}

	// store data
	o.token = data.Token
	o.refresh = time.Now().Add(time.Duration(data.ExpiresIn) * time.Second)
	if data.RefreshToken != "" {
		o.refreshToken = data.RefreshToken
	}
	log.Printf("oauth2: stored token, expires on %s", o.refresh)
	return nil
}

func (o *OAuth2) checkTokenExpiration() error {
	if time.Until(o.refresh) > 0 {
		return nil
	}

	o.refreshLock.Lock()
	defer o.refreshLock.Unlock()

	if time.Until(o.refresh) > 0 {
		return nil
	}

	if o.refreshToken == "" {
		return errors.New("session has expired, please login again")
	}

	log.Printf("oauth2: refreshing token")

	// perform refresh
	resp, err := http.PostForm(o.endpoint, url.Values{"grant_type": {"refresh_token"}, "client_id": {o.clientId}, "refresh_token": {o.refreshToken}})
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return o.storeToken(body)
}

func (o *OAuth2) RoundTrip(r *http.Request) (*http.Response, error) {
	err := o.checkTokenExpiration()
	if err != nil {
		return nil, err
	}

	r.Header.Add("Authorization", "Bearer "+o.token)
	return o.Transport.RoundTrip(r)
}

type RestParam map[string]interface{}

type RestResponse struct {
	Result string      `json:"result"` // "success" or "error" (or "redirect")
	Data   interface{} `json:"data"`
	Error  string      `json:"error"`

	Paging interface{} `json:"paging"`
	Job    interface{} `json:"job"`
	Time   interface{} `json:"time"`
	Access interface{} `json:"access"`

	RedirectUrl  string `json:"redirect_url"`
	RedirectCode int    `json:"redirect_code"`
}

func (o *OAuth2) Rest(req, method string, param RestParam) (*RestResponse, error) {
	// build http request
	r := &http.Request{
		Method: method,
		URL: &url.URL{
			Scheme: "https",
			Host:   "www.atonline.com",
			Path:   "/_special/rest/" + req,
		},
		Header: make(http.Header),
	}

	r.Header.Set("Sec-Rest-Http", "false")

	// add parameters (depending on method)
	switch method {
	case "GET", "HEAD", "OPTIONS":
		// need to pass parameters in GET
		r.URL.RawQuery = util.EncodePhpQuery(param)
	case "POST", "PATCH":
		data, err := json.Marshal(param)
		if err != nil {
			return nil, err
		}
		buf := bytes.NewReader(data)
		r.Body = ioutil.NopCloser(buf)
		r.ContentLength = int64(len(data))
		r.GetBody = func() (io.ReadCloser, error) {
			reader := bytes.NewReader(data)
			return ioutil.NopCloser(reader), nil
		}
		r.Header.Set("Content-Type", "application/json")
	case "DELETE":
		// nothing
	default:
		return nil, fmt.Errorf("invalid request method %s", method)
	}

	t := time.Now()

	resp, err := o.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	d := time.Since(t)
	log.Printf("[rest] %s %s => %s", method, req, d)

	//util.CtxPrintf(ctx, "[debug] Response to %s %s: %s", method, req, body)

	result := &RestResponse{}
	err = json.Unmarshal(body, result)
	if err != nil {
		log.Printf("failed to parse json: %s %s", err, body)
		return nil, err
	}

	if result.Result == "redirect" {
		url, err := url.Parse(result.RedirectUrl)
		if err != nil {
			return nil, err
		}
		return nil, RedirectErrorCode(url, result.RedirectCode)
	}

	if result.Result == "error" {
		return nil, fmt.Errorf("[rest] error from server: %s", result.Error)
	}

	return result, nil
}
