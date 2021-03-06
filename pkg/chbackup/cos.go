package chbackup

import (
	"context"
	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/tencentyun/cos-go-sdk-v5/debug"
	"io"
	"net/http"
	"net/url"
	"time"
)

type COS struct {
	client *cos.Client
	Config *COSConfig
}

// Connect - connect to cos
func (c *COS) Connect() error {
	u, _ := url.Parse(c.Config.RowUrl)
	b := &cos.BaseURL{BucketURL: u}
	c.client = cos.NewClient(b, &http.Client{
		Timeout: time.Duration(c.Config.Timeout) * time.Second,
		Transport: &cos.AuthorizationTransport{
			SecretID:  c.Config.SecretID,
			SecretKey: c.Config.SecretKey,
			// request debug
			Transport: &debug.DebugRequestTransport{
				RequestHeader:  c.Config.Debug,
				RequestBody:    false,
				ResponseHeader: c.Config.Debug,
				ResponseBody:   false,
			},
		},
	})
	// check bucket exists
	_, err := c.client.Bucket.Head(context.Background())
	if err != nil {
		return err
	}
	return nil
}

func (c *COS) Kind() string {
	return "COS"
}

func (c *COS) GetFile(key string) (RemoteFile, error) {
	// file max size is 5Gb
	resp, err := c.client.Object.Get(context.Background(), key, nil)
	if err != nil {
		cosErr,ok := err.(*cos.ErrorResponse)
		if ok && cosErr.Code == "NoSuchKey" {
			return nil, ErrNotFound
		}
		return nil, err
	}
	modifiedTime,_ := parseTime(resp.Response.Header.Get("Date"))
	return &cosFile{
		size:resp.Response.ContentLength,
		name:resp.Request.URL.Path,
		lastModified: modifiedTime,
	}, nil
}

func (c *COS) DeleteFile(key string) error {
	_, err := c.client.Object.Delete(context.Background(), key)
	if err != nil {
		return err
	}
	return nil
}

func (c *COS) Walk(path string, process func(RemoteFile)) error {
	res, _, err := c.client.Bucket.Get(context.Background(), &cos.BucketGetOptions{
		Prefix: c.Config.Path,
	})
	if err != nil {
		return err
	}
	for _, v := range res.Contents {
		modifiedTime,_ := parseTime(v.LastModified)
		process(&cosFile{
			name:v.Key,
			lastModified: modifiedTime,
			size: int64(v.Size),
		})
	}
	return nil
}

func (c *COS) GetFileReader(key string) (io.ReadCloser, error) {
	resp, err := c.client.Object.Get(context.Background(), key, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (c *COS) PutFile(key string, r io.ReadCloser) error {
	_, err := c.client.Object.Put(context.Background(), key, r, nil)
	if err != nil {
		return err
	}
	return nil
}

type cosFile struct {
	size         int64
	lastModified time.Time
	name         string
}

func (f *cosFile) Size() int64 {
	return f.size
}

func (f *cosFile) Name() string {
	return f.name
}

func (f *cosFile) LastModified() time.Time {
	return f.lastModified
}


