// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Client encapsulates the S3 actions used by the rgw command line tool
// against a RADOS Gateway / S3-compatible store.
type Client struct {
	S3Client *s3.Client
}

// New loads the default AWS configuration and returns a Client backed by an
// S3 client. Endpoint and credentials are resolved from the environment (for
// example AWS_ENDPOINT_URL, AWS_REGION, and the standard credential chain).
// Rados gateways typically lack wildcard DNS for virtual-hosted addressing
// (<bucket>.<endpoint>), so path-style addressing is forced.
func New(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	return &Client{S3Client: s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})}, nil
}

// CreateBucket creates a bucket, optionally in a specific region. An empty
// LocationConstraint is rejected by RGW, so the configuration is omitted
// entirely when no region is given.
func (c Client) CreateBucket(ctx context.Context, name string, region string) error {
	input := &s3.CreateBucketInput{Bucket: aws.String(name)}
	if region != "" {
		input.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}
	if _, err := c.S3Client.CreateBucket(ctx, input); err != nil {
		return fmt.Errorf("creating bucket %q: %w", name, err)
	}
	if err := s3.NewBucketExistsWaiter(c.S3Client).Wait(
		ctx, &s3.HeadBucketInput{Bucket: aws.String(name)}, time.Minute); err != nil {
		return fmt.Errorf("waiting for bucket %q to exist: %w", name, err)
	}
	return nil
}

// DeleteBucket deletes a bucket. The bucket must be empty or an error is returned.
func (c Client) DeleteBucket(ctx context.Context, bucketName string) error {
	if _, err := c.S3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName)}); err != nil {
		var noBucket *types.NoSuchBucket
		if errors.As(err, &noBucket) {
			return fmt.Errorf("bucket %q does not exist: %w", bucketName, err)
		}
		return fmt.Errorf("deleting bucket %q: %w", bucketName, err)
	}
	if err := s3.NewBucketNotExistsWaiter(c.S3Client).Wait(
		ctx, &s3.HeadBucketInput{Bucket: aws.String(bucketName)}, time.Minute); err != nil {
		return fmt.Errorf("waiting for bucket %q to be deleted: %w", bucketName, err)
	}
	return nil
}

// ListBuckets lists the buckets in the current account.
func (c Client) ListBuckets(ctx context.Context) ([]types.Bucket, error) {
	var buckets []types.Bucket
	paginator := s3.NewListBucketsPaginator(c.S3Client, &s3.ListBucketsInput{})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing buckets: %w", err)
		}
		buckets = append(buckets, out.Buckets...)
	}
	return buckets, nil
}

// DeleteObject deletes an object from a bucket.
func (c Client) DeleteObject(ctx context.Context, bucket string, key string) error {
	if _, err := c.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}); err != nil {
		var noKey *types.NoSuchKey
		if errors.As(err, &noKey) {
			return fmt.Errorf("object %s does not exist in bucket %s: %w", key, bucket, err)
		}
		return fmt.Errorf("deleting object %v:%v: %w", bucket, key, err)
	}
	if err := s3.NewObjectNotExistsWaiter(c.S3Client).Wait(
		ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)}, time.Minute); err != nil {
		return fmt.Errorf("waiting for object %s to be deleted: %w", key, err)
	}
	return nil
}

// DownloadFile gets an object from a bucket and streams it to a local file.
func (c Client) DownloadFile(ctx context.Context, bucketName string, objectKey string, fileName string) error {
	result, err := c.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("downloading %v:%v: %w", bucketName, objectKey, err)
	}
	defer result.Body.Close()

	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("creating local file %v: %w", fileName, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, result.Body); err != nil {
		return fmt.Errorf("writing %v: %w", fileName, err)
	}
	return nil
}

// ListObjects lists the objects in a bucket.
func (c Client) ListObjects(ctx context.Context, bucketName string) ([]types.Object, error) {
	var objects []types.Object
	paginator := s3.NewListObjectsV2Paginator(c.S3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing objects in %q: %w", bucketName, err)
		}
		objects = append(objects, out.Contents...)
	}
	return objects, nil
}

// UploadFile streams a local file into a bucket via the multipart-capable
// transfer manager: large tarballs are split into parts automatically and
// single-request size limits do not apply.
func (c Client) UploadFile(ctx context.Context, bucketName string, objectKey string, fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("opening %v for upload: %w", fileName, err)
	}
	defer file.Close()

	uploader := manager.NewUploader(c.S3Client)
	if _, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   file,
	}); err != nil {
		return fmt.Errorf("uploading %v to %v:%v: %w", fileName, bucketName, objectKey, err)
	}
	return nil
}
