package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
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
