package s3

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/blockdaemon/s3-bucket-browser/internal/config"
)

// Service represents the S3 service
type Service struct {
	client *s3.Client
	bucket string
}

// NewService creates a new S3 service
func NewService(cfg *config.S3Config) (*Service, error) {
	// Create AWS configuration
	awsCfg, err := createAWSConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)

	return &Service{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// createAWSConfig creates an AWS configuration
func createAWSConfig(cfg *config.S3Config) (aws.Config, error) {
	options := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	}

	// Use custom endpoint if provided
	if cfg.Endpoint != "" {
		options = append(options, awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:               cfg.Endpoint,
						SigningRegion:     cfg.Region,
						HostnameImmutable: true,
					}, nil
				},
			),
		))
	}

	return awsconfig.LoadDefaultConfig(context.Background(), options...)
}

// ListObjects lists objects in the S3 bucket
func (s *Service) ListObjects(ctx context.Context, prefix string) ([]Object, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	}

	result, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, err
	}

	objects := make([]Object, 0, len(result.Contents))
	for _, obj := range result.Contents {
		size := int64(0)
		if obj.Size != nil {
			size = *obj.Size
		}

		objects = append(objects, Object{
			Key:          *obj.Key,
			Size:         size,
			LastModified: *obj.LastModified,
			ETag:         *obj.ETag,
			IsTarGz:      IsTarGzFile(*obj.Key),
			IsMetadata:   strings.HasSuffix(*obj.Key, ".json"),
		})
	}

	return objects, nil
}

// GetObject gets an object from the S3 bucket
func (s *Service) GetObject(ctx context.Context, key string) (*s3.GetObjectOutput, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	return s.client.GetObject(ctx, input)
}

// IsTarGzFile checks if a file is a .tar.gz file
func IsTarGzFile(key string) bool {
	return strings.HasSuffix(key, ".tar.gz")
}

// GetMetadataFileKey returns the metadata file key for a .tar.gz file
func GetMetadataFileKey(tarGzKey string) string {
	// Remove .tar.gz extension and add .json
	return strings.TrimSuffix(tarGzKey, ".tar.gz") + ".json"
}

// Object represents an S3 object
type Object struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
	IsMetadata   bool
	IsTarGz      bool
}
