package oauth2

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

const uploadBlockLen = 5 * 1024 * 1024
const awsTimeFormat = "20060102T150405Z"

type Upload struct {
	o           *OAuth2
	buf         *bytes.Buffer
	pos         int64
	committed   int64 // sent so far
	maxLen      int
	ContentType string

	chunks []string

	upload   map[string]interface{}
	upid     string
	putUrl   string
	complete string
	uploadId string // AWS upload id

	bucketHost string
	bucketName string
	region     string
	key        string
	awsUrl     string // combined of previous things
}

func NewUpload(o *OAuth2, req string, param RestParam) (*Upload, error) {
	apires, err := o.Rest(req, "POST", param)
	if err != nil {
		return nil, err
	}

	data := apires.Data.(map[string]interface{})

	// we should be getting some data
	res := &Upload{o: o, upload: data, maxLen: uploadBlockLen, buf: &bytes.Buffer{}}
	res.upid = data["Cloud_Aws_Bucket_Upload__"].(string)
	res.putUrl = data["PUT"].(string)
	res.complete = data["Complete"].(string)

	bucket := data["Bucket_Endpoint"].(map[string]interface{})
	res.bucketHost = bucket["Host"].(string)
	res.bucketName = bucket["Name"].(string)
	res.region = bucket["Region"].(string)
	res.key = data["Key"].(string)
	res.awsUrl = "https://" + res.bucketHost + "/" + res.bucketName + "/" + res.key

	return res, nil
}

func (u *Upload) Len() int64 {
	return u.pos
}

func (u *Upload) Complete() (*RestResponse, error) {
	// finalize upload
	if len(u.chunks) == 0 {
		// perform regular PUT upload
		log.Printf("Performing PUT upload (%d bytes)", u.buf.Len())
		req, err := http.NewRequest("PUT", u.putUrl, bytes.NewReader(u.buf.Bytes()))
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
	} else {
		if u.buf.Len() > 0 {
			err := u.sendBlock()
			if err != nil {
				return nil, err
			}
		}

		// need to finalize upload with AWS, passing all chunk ids
		xml := &bytes.Buffer{}
		xml.Write([]byte("<CompleteMultipartUpload>"))
		for i, c := range u.chunks {
			fmt.Fprintf(xml, "<Part><PartNumber>%d</PartNumber><ETag>%s</ETag></Part>", i+1, c)
		}
		xml.Write([]byte("</CompleteMultipartUpload>"))

		req, err := http.NewRequest("POST", u.awsUrl+"?uploadId="+url.QueryEscape(u.uploadId), nil)
		if err != nil {
			return nil, err
		}

		// perform AWS request
		resp, err := u.awsReq(req, xml.Bytes())
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			// may need to include body
			return nil, fmt.Errorf("AWS request failed: %s", resp.Status)
		}
	}

	// perform finalize
	return u.o.Rest(u.complete, "POST", nil)
}

func (u *Upload) Write(d []byte) (int, error) {
	e, err := u.buf.Write(d)
	if e > 0 {
		u.pos += int64(e)
	}
	if err != nil {
		return e, err
	}
	if u.buf.Len() >= u.maxLen {
		err = u.sendBlock()
		u.maxLen += uploadBlockLen
	}
	return e, err
}

func (u *Upload) sendBlock() error {
	// flush buffer now
	buf := u.buf
	u.committed += int64(buf.Len())
	u.buf = &bytes.Buffer{}
	partId := len(u.chunks) + 1
	log.Printf("Performing chunk upload (%d bytes)", buf.Len())

	if u.ContentType == "" {
		// need to guess content type
		// or we could set it to application/octet-stream and let the platform do the job
		u.ContentType = http.DetectContentType(buf.Bytes())
	}

	if u.uploadId == "" {
		// need to initialize upload with aws
		req, err := http.NewRequest("POST", u.awsUrl+"?uploads=", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", u.ContentType)
		req.Header.Set("X-Amz-Acl", "private")
		resp, err := u.awsReq(req, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var v awsInitiateMultipartUploadResult
		err = xml.Unmarshal(body, &v)
		if err != nil {
			return err
		}

		u.uploadId = v.UploadId
	}

	// ok now we need to upload this part
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s?partNumber=%d&uploadId=%s", u.awsUrl, partId, url.QueryEscape(u.uploadId)), nil)
	resp, err := u.awsReq(req, buf.Bytes())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// may need to include body
		return fmt.Errorf("AWS request failed: %s", resp.Status)
	}

	u.chunks = append(u.chunks, resp.Header.Get("ETag"))

	return nil
}
