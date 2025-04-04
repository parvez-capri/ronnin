package services

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

// S3Service handles uploading files to AWS S3
type S3Service struct {
	client     *s3.Client
	bucketName string
	region     string
	baseURL    string
	presigner  *s3.PresignClient
}

// NewS3Service creates a new S3 service instance
func NewS3Service(accessKey, secretKey, region, bucketName, baseURL string) (*S3Service, error) {
	// Create AWS credentials
	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")

	// Configure AWS SDK
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)
	// Create presigner client
	presigner := s3.NewPresignClient(client)

	return &S3Service{
		client:     client,
		presigner:  presigner,
		bucketName: bucketName,
		region:     region,
		baseURL:    baseURL,
	}, nil
}

// UploadFile uploads a file to S3 and returns a presigned URL with 7 days expiry
func (s *S3Service) UploadFile(ctx context.Context, file *multipart.FileHeader) (string, error) {
	fmt.Printf("\n=== S3 UPLOAD ATTEMPT ===\n")
	fmt.Printf("Filename: %s\n", file.Filename)
	fmt.Printf("File size: %d bytes\n", file.Size)
	fmt.Printf("Content type: %s\n", file.Header.Get("Content-Type"))

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		fmt.Printf("ERROR: Failed to open uploaded file: %s\n", err)
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Read file content
	buffer := make([]byte, file.Size)
	bytesRead, err := src.Read(buffer)
	if err != nil {
		fmt.Printf("ERROR: Failed to read file content: %s\n", err)
		return "", fmt.Errorf("failed to read file content: %w", err)
	}
	fmt.Printf("Bytes read: %d\n", bytesRead)

	// Create a unique key for the file
	fileExt := filepath.Ext(file.Filename)
	objectKey := fmt.Sprintf("uploads/ronnin/%s%s", uuid.New().String(), fileExt)
	fmt.Printf("Generated S3 object key: %s\n", objectKey)
	fmt.Printf("Target bucket: %s\n", s.bucketName)
	fmt.Printf("Region: %s\n", s.region)

	// Upload to S3
	putObjectOutput, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(buffer),
		ContentType: aws.String(file.Header.Get("Content-Type")),
		ACL:         types.ObjectCannedACLPrivate,
	})

	if err != nil {
		fmt.Printf("ERROR: S3 upload failed: %s\n", err)
		fmt.Printf("=== END S3 UPLOAD (FAILED) ===\n\n")
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	fmt.Printf("S3 PutObject successful\n")
	fmt.Printf("Response ETag: %s\n", putObjectOutput.ETag)

	// Generate presigned URL with 7-day expiry
	presignDuration := time.Hour * 24 * 7 // 7 days
	presignedReq, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = presignDuration
	})

	if err != nil {
		fmt.Printf("ERROR: Failed to generate presigned URL: %s\n", err)

		// Fall back to regular URL if presigning fails
		var fileURL string
		if s.baseURL != "" {
			fileURL = fmt.Sprintf("%s/%s", s.baseURL, objectKey)
		} else {
			fileURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, objectKey)
		}

		fmt.Printf("WARNING: Using non-presigned URL as fallback: %s\n", fileURL)
		fmt.Printf("=== END S3 UPLOAD (PARTIAL SUCCESS) ===\n\n")
		return fileURL, nil
	}

	// Log and return the presigned URL
	fmt.Printf("Generated presigned URL (expires in 7 days): %s\n", presignedReq.URL)
	fmt.Printf("=== END S3 UPLOAD (SUCCESS) ===\n\n")

	return presignedReq.URL, nil
}
