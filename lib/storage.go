package lib

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// AWSSession returns a new AWS SDK session object based on environment variables.
func AWSSession(prefix string) *session.Session {
	config := &aws.Config{}
	if v := Env(prefix+"_REGION", ""); v != "" {
		config.Region = aws.String(v)
	} else {
		config.Region = aws.String("us-east-1")
	}
	if v := Env(prefix+"_URL", ""); v != "" {
		config.Endpoint = aws.String(v)
	}
	if Env(prefix+"_ACCESS_KEY", "") != "" {
		config.Credentials = credentials.NewStaticCredentials(Env(prefix+"_ACCESS_KEY", ""), Env(prefix+"_SECRET_KEY", ""), "")
	}
	return session.Must(session.NewSession(config))
}

// AWSS3Key transforms an entity id into an S3 key (prefixing 2 short folder to it for proper sharding)
func AWSS3Key(key string) string {
	return fmt.Sprintf("%s/%s/%s", key[len(key)-4:len(key)-2], key[len(key)-2:], key)
}

// AWSS3URL returns the url for a given entity id
func AWSS3URL(key string) string {
	return fmt.Sprintf("%s/%s/%s", Env("S3_URL", "s3.us-east-1.amazonaws.com"), Env("S3_BUCKET", ""), AWSS3Key(key))
}

// Storage represents an instance of a storage service
type Storage struct {
	ctx      *Ctx
	public   bool
	bucket   string
	client   *s3.S3
	uploader *s3manager.Uploader
}

// NewStorage returns a new instance of Storage for a given bucket and acl
func NewStorage(bucket string, public bool) *Storage {
	s := AWSSession("S3")
	client := s3.New(s)
	uploader := s3manager.NewUploader(s)
	return &Storage{public: public, bucket: bucket,
		client: client, uploader: uploader}
}

func (s *Storage) WithCtx(ctx *Ctx) *Storage {
	return &Storage{ctx: ctx, public: s.public, bucket: s.bucket,
		client: s.client, uploader: s.uploader}
}

func (s *Storage) acl() string {
	if s.public {
		return "public-read"
	}
	return "private"
}

// Get retrieves the value of an object in storage
func (s *Storage) Get(key string) []byte {
	bs, err := s.GetErr(key)
	Check(err)
	return bs
}

// GetErr retrieves the value of an object in storage
func (s *Storage) GetErr(key string) ([]byte, error) {
	result, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(AWSS3Key(key)),
	})
	Check(err)
	defer result.Body.Close()
	bs, err := ioutil.ReadAll(result.Body)
	return bs, err
}

// Set stores the given value in storage
func (s *Storage) Set(key string, value []byte) {
	Check(s.SetErr(key, value))
}

// SetErr stores the given value in storage
func (s *Storage) SetErr(key string, value []byte) error {
	typ := http.DetectContentType(value)
	_, err := s.client.PutObject(&s3.PutObjectInput{
		ACL:           aws.String(s.acl()),
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(AWSS3Key(key)),
		Body:          bytes.NewReader(value),
		ContentType:   aws.String(typ),
		ContentLength: aws.Int64(int64(len(value))),
	})
	return err
}

// PutStreaming stores the given value in storage, streaming. (Using an io.Reader instead of []byte)
func (s *Storage) PutStreaming(key string, contentType string, r io.Reader) {
	_, err := s.uploader.Upload(&s3manager.UploadInput{
		ACL:         aws.String(s.acl()),
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(AWSS3Key(key)),
		Body:        r,
		ContentType: aws.String(contentType),
	})
	Check(err)
}

// Delete removes the value at the given key
func (s *Storage) Delete(key string) {
	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(AWSS3Key(key)),
	})
	Check(err)
}

// Size returns the size of the object for the given key
func (s *Storage) Size(key string) int64 {
	head, err := s.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(AWSS3Key(key)),
	})
	Check(err)
	var length int64
	if head != nil && head.ContentLength != nil {
		length = *head.ContentLength
	}
	return length
}

// SignedURL returns a signed URL allowing anybody to download the object for the given key
func (s *Storage) SignedURL(key, filename string) string {
	req, _ := s.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket:                     aws.String(s.bucket),
		Key:                        aws.String(AWSS3Key(key)),
		ResponseContentDisposition: aws.String("filename =\"" + filename + "\""),
	})
	url, err := req.Presign(30 * time.Minute)
	Check(err)
	return url
}

// SignedPutURL retuns a signed URL that can be used to upload an object for the given key
func (s *Storage) SignedPutURL(key, contentType string) (string, http.Header) {
	req, _ := s.client.PutObjectRequest(&s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(AWSS3Key(key)),
		ACL:         aws.String(s.acl()),
		ContentType: aws.String(contentType),
	})
	url, httpHeaders, err := req.PresignRequest(30 * time.Minute)
	Check(err)
	return url, httpHeaders
}
