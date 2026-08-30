// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const xmlNS = `xmlns="http://s3.amazonaws.com/doc/2006-03-01/"`

// fakeS3 is a minimal in-memory S3 server speaking just enough of the API
// (path-style addressing, PutObject, GetObject, ListObjectsV2, buckets, and
// the multipart upload flow) to exercise the Client offline.
type fakeS3 struct {
	mu        sync.Mutex
	buckets   map[string]map[string][]byte // bucket -> key -> bytes
	uploads   map[string]*multipartState   // uploadID -> in-progress upload
	nextID    int
	partCalls int
}

type multipartState struct {
	bucket string
	key    string
	parts  map[int][]byte
}

func newFakeS3() *fakeS3 {
	return &fakeS3{
		buckets: make(map[string]map[string][]byte),
		uploads: make(map[string]*multipartState),
	}
}

func writeXML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	fmt.Fprint(w, body)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeXML(w, status, fmt.Sprintf(`<Error %s><Code>%s</Code><Message>%s</Message></Error>`, xmlNS, code, code))
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	p := strings.TrimPrefix(r.URL.Path, "/")
	bucket, key, _ := strings.Cut(p, "/")
	q := r.URL.Query()
	uploadID := q.Get("uploadId")

	switch {
	case p == "" && r.Method == http.MethodGet:
		f.listBuckets(w)

	case key == "":
		f.handleBucket(w, r, bucket)

	case r.Method == http.MethodPost && q.Has("uploads"):
		f.nextID++
		id := strconv.Itoa(f.nextID)
		f.uploads[id] = &multipartState{bucket: bucket, key: key, parts: map[int][]byte{}}
		writeXML(w, http.StatusOK, fmt.Sprintf(
			`<InitiateMultipartUploadResult %s><Bucket>%s</Bucket><Key>%s</Key><UploadId>%s</UploadId></InitiateMultipartUploadResult>`,
			xmlNS, bucket, key, id))

	case r.Method == http.MethodPost && uploadID != "":
		f.completeMultipart(w, uploadID)

	case r.Method == http.MethodPut && uploadID != "" && q.Get("partNumber") != "":
		n, _ := strconv.Atoi(q.Get("partNumber"))
		up, ok := f.uploads[uploadID]
		if !ok {
			writeError(w, http.StatusNotFound, "NoSuchUpload")
			return
		}
		body, _ := io.ReadAll(r.Body)
		up.parts[n] = body
		f.partCalls++
		w.Header().Set("ETag", fmt.Sprintf(`"etag-%s-%d"`, uploadID, n))
		w.WriteHeader(http.StatusOK)

	case r.Method == http.MethodPut:
		f.putObject(w, bucket, key, r.Body)
	case r.Method == http.MethodGet:
		f.getObject(w, bucket, key)
	case r.Method == http.MethodHead:
		f.headObject(w, bucket, key)
	case r.Method == http.MethodDelete:
		delete(f.buckets[bucket], key)
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusNotImplemented, "NotImplemented")
	}
}

func (f *fakeS3) handleBucket(w http.ResponseWriter, r *http.Request, bucket string) {
	switch r.Method {
	case http.MethodPut:
		f.buckets[bucket] = map[string][]byte{}
		w.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		if _, ok := f.buckets[bucket]; !ok {
			writeError(w, http.StatusNotFound, "NoSuchBucket")
			return
		}
		delete(f.buckets, bucket)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodHead:
		if _, ok := f.buckets[bucket]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		f.listObjects(w, bucket)
	default:
		writeError(w, http.StatusNotImplemented, "NotImplemented")
	}
}

func (f *fakeS3) listBuckets(w http.ResponseWriter) {
	var b strings.Builder
	for name := range f.buckets {
		fmt.Fprintf(&b, `<Bucket><Name>%s</Name><CreationDate>2026-01-01T00:00:00Z</CreationDate></Bucket>`, name)
	}
	writeXML(w, http.StatusOK, fmt.Sprintf(
		`<ListAllMyBucketsResult %s><Owner><ID>fake</ID></Owner><Buckets>%s</Buckets></ListAllMyBucketsResult>`, xmlNS, b.String()))
}

func (f *fakeS3) listObjects(w http.ResponseWriter, bucket string) {
	objs, ok := f.buckets[bucket]
	if !ok {
		writeError(w, http.StatusNotFound, "NoSuchBucket")
		return
	}
	var b strings.Builder
	keys := make([]string, 0, len(objs))
	for k := range objs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, `<Contents><Key>%s</Key><Size>%d</Size><ETag>"etag"</ETag><StorageClass>STANDARD</StorageClass></Contents>`, k, len(objs[k]))
	}
	writeXML(w, http.StatusOK, fmt.Sprintf(
		`<ListBucketResult %s><Name>%s</Name>%s<KeyCount>%d</KeyCount></ListBucketResult>`, xmlNS, bucket, b.String(), len(keys)))
}

func (f *fakeS3) putObject(w http.ResponseWriter, bucket, key string, body io.Reader) {
	data, _ := io.ReadAll(body)
	if _, ok := f.buckets[bucket]; !ok {
		f.buckets[bucket] = map[string][]byte{}
	}
	f.buckets[bucket][key] = data
	w.Header().Set("ETag", `"etag"`)
	w.WriteHeader(http.StatusOK)
}

func (f *fakeS3) completeMultipart(w http.ResponseWriter, uploadID string) {
	up, ok := f.uploads[uploadID]
	if !ok {
		writeError(w, http.StatusNotFound, "NoSuchUpload")
		return
	}
	delete(f.uploads, uploadID)
	nums := make([]int, 0, len(up.parts))
	for n := range up.parts {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	var data []byte
	for _, n := range nums {
		data = append(data, up.parts[n]...)
	}
	if _, ok := f.buckets[up.bucket]; !ok {
		f.buckets[up.bucket] = map[string][]byte{}
	}
	f.buckets[up.bucket][up.key] = data
	writeXML(w, http.StatusOK, fmt.Sprintf(
		`<CompleteMultipartUploadResult %s><Location>http://fake/%s/%s</Location><Bucket>%s</Bucket><Key>%s</Key><ETag>"etag-final"</ETag></CompleteMultipartUploadResult>`,
		xmlNS, up.bucket, up.key, up.bucket, up.key))
}

func (f *fakeS3) getObject(w http.ResponseWriter, bucket, key string) {
	data, ok := f.buckets[bucket][key]
	if !ok {
		writeError(w, http.StatusNotFound, "NoSuchKey")
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (f *fakeS3) headObject(w http.ResponseWriter, bucket, key string) {
	data, ok := f.buckets[bucket][key]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
}

// newTestClient returns a Client pointed at an in-memory fake S3 server.
func newTestClient(t *testing.T) (Client, *fakeS3) {
	t.Helper()
	fake := newFakeS3()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	return Client{S3Client: s3.New(s3.Options{
		Region:       "us-east-1",
		Credentials:  aws.AnonymousCredentials{},
		BaseEndpoint: aws.String(srv.URL),
		UsePathStyle: true,
	})}, fake
}

func writeTempFile(t *testing.T, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "blob")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return p
}

func TestBucketLifecycle(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestClient(t)

	if err := c.CreateBucket(ctx, "archives", ""); err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}
	buckets, err := c.ListBuckets(ctx)
	if err != nil {
		t.Fatalf("ListBuckets: %v", err)
	}
	if len(buckets) != 1 || *buckets[0].Name != "archives" {
		t.Fatalf("ListBuckets = %v, want single bucket \"archives\"", buckets)
	}
	if err := c.DeleteBucket(ctx, "archives"); err != nil {
		t.Fatalf("DeleteBucket: %v", err)
	}
	buckets, err = c.ListBuckets(ctx)
	if err != nil {
		t.Fatalf("ListBuckets after delete: %v", err)
	}
	if len(buckets) != 0 {
		t.Errorf("ListBuckets after delete = %v, want empty", buckets)
	}
}

func TestUploadDownloadRoundTrip(t *testing.T) {
	ctx := context.Background()
	c, fake := newTestClient(t)

	if err := c.CreateBucket(ctx, "archives", ""); err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}
	content := []byte(strings.Repeat("small-payload-", 100))
	if err := c.UploadFile(ctx, "archives", "a/b/small.bin", writeTempFile(t, content)); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if fake.partCalls != 0 {
		t.Errorf("small upload used %d multipart part calls, want 0 (single PutObject)", fake.partCalls)
	}

	dst := filepath.Join(t.TempDir(), "out")
	if err := c.DownloadFile(ctx, "archives", "a/b/small.bin", dst); err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading download: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("round trip mismatch: got %d bytes, want %d", len(got), len(content))
	}
}

func TestUploadLargeUsesMultipart(t *testing.T) {
	ctx := context.Background()
	c, fake := newTestClient(t)

	if err := c.CreateBucket(ctx, "archives", ""); err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}
	// 12 MiB + offset: exceeds the 5 MiB default part size
	content := bytes.Repeat([]byte("multipart-"), 12*1024*1024/10+13)
	if err := c.UploadFile(ctx, "archives", "big.tar.gz", writeTempFile(t, content)); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if fake.partCalls < 2 {
		t.Errorf("expected at least 2 multipart part calls for a >5 MiB file, got %d", fake.partCalls)
	}

	dst := filepath.Join(t.TempDir(), "out")
	if err := c.DownloadFile(ctx, "archives", "big.tar.gz", dst); err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading download: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("multipart round trip mismatch: got %d bytes, want %d", len(got), len(content))
	}
}

func TestListObjects(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestClient(t)

	if err := c.CreateBucket(ctx, "archives", ""); err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}
	for _, key := range []string{"one.tar.gz", "two.tar.gz"} {
		if err := c.UploadFile(ctx, "archives", key, writeTempFile(t, []byte("x"))); err != nil {
			t.Fatalf("UploadFile %s: %v", key, err)
		}
	}
	objects, err := c.ListObjects(ctx, "archives")
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("ListObjects returned %d objects, want 2", len(objects))
	}
	if *objects[0].Key != "one.tar.gz" || *objects[1].Key != "two.tar.gz" {
		t.Errorf("ListObjects keys = %q, %q; want one.tar.gz, two.tar.gz", *objects[0].Key, *objects[1].Key)
	}
	if _, err := c.ListObjects(ctx, "missing-bucket"); err == nil {
		t.Error("expected error listing a missing bucket")
	}
}

func TestObjectDeleteAndMissingKey(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestClient(t)

	if err := c.CreateBucket(ctx, "archives", ""); err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}
	key := "doomed.tar.gz"
	if err := c.UploadFile(ctx, "archives", key, writeTempFile(t, []byte("x"))); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if err := c.DeleteObject(ctx, "archives", key); err != nil {
		t.Fatalf("DeleteObject: %v", err)
	}
	err := c.DownloadFile(ctx, "archives", key, filepath.Join(t.TempDir(), "out"))
	if err == nil || !strings.Contains(err.Error(), "NoSuchKey") {
		t.Errorf("downloading a deleted object: err = %v, want NoSuchKey", err)
	}
}
