package oauth2

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type awsInitiateMultipartUploadResult struct {
	Bucket   string
	Key      string
	UploadId string
}

var awsClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{

			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true, // required for AWS
	},
}

func (u *Upload) awsReq(req *http.Request, body []byte) (*http.Response, error) {
	// perform aws request
	bodyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // sha256('')
	if body != nil {
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
		req.GetBody = func() (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewReader(body)), nil
		}
		req.ContentLength = int64(len(body))
		hash := sha256.New()
		hash.Write(body)
		sum := hash.Sum(nil)
		bodyHash = hex.EncodeToString(sum)
	}

	ts := time.Now().UTC().Format(awsTimeFormat)
	tsD := ts[:8] // date part, YYYYMMDD

	req.Header.Set("X-Amz-Content-Sha256", bodyHash)
	req.Header.Set("X-Amz-Date", ts)
	req.TransferEncoding = []string{"identity"} // should be OK anyway since we set ContentLength, but just in case

	awsAuthStr := []string{
		"AWS4-HMAC-SHA256",
		ts,
		tsD + "/" + u.region + "/s3/aws4_request",
		req.Method,
		req.URL.Path,
		req.URL.RawQuery,
		"host:" + req.URL.Host,
	}

	sign_head := []string{"host"}
	var k sort.StringSlice
	for h := range req.Header {
		k = append(k, h)
	}
	k.Sort()

	for _, h := range k {
		s := strings.ToLower(h)
		if !strings.HasPrefix(s, "x-") {
			continue
		}
		sign_head = append(sign_head, s)
		awsAuthStr = append(awsAuthStr, s+":"+req.Header.Get(h))
	}
	awsAuthStr = append(awsAuthStr, "", strings.Join(sign_head, ";"), bodyHash)

	// need to ask platform to sign this
	signRes, err := u.o.Rest("Cloud/Aws/Bucket/Upload/"+url.PathEscape(u.upid)+":signV4", "POST", RestParam{"headers": strings.Join(awsAuthStr, "\n")})
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", signRes.Data.(map[string]interface{})["authorization"].(string))

	return awsClient.Do(req)
}
