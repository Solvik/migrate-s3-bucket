package main

import (
	"fmt"
	"flag"
	"bufio"
	"io/ioutil"
	"os"
	"bytes"
	"sync"
	"strings"
        "time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"gopkg.in/yaml.v2"
)

// Define structs to match your YAML configuration
type Config struct {
	Profiles struct {
		OldProfile ProfileConfig `yaml:"oldProfile"`
		NewProfile ProfileConfig `yaml:"newProfile"`
	} `yaml:"profiles"`
}

type ProfileConfig struct {
	Region      string `yaml:"region"`
	Endpoint    string `yaml:"endpoint"`
	AccessKey   string `yaml:"accessKey"`
	SecretKey   string `yaml:"secretKey"`
}

// ReadConfig reads the YAML configuration file and unmarshals it into a Config struct
func ReadConfig(configPath string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}
	var config Config
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}
	return &config, nil
}

var (
	bucket	   string
	configPath string
	filename   string
	checkExistence bool
)

func init() {
	flag.StringVar(&configPath, "config", "", "Path to the configuration file")
	flag.StringVar(&filename, "filename", "", "Path to the file containing keys to migrate")
	flag.StringVar(&bucket, "bucket", "", "Name of the S3 bucket to operate on") // Initialize the bucket flag
	flag.BoolVar(&checkExistence, "check", false, "Check if the file exists in the destination bucket before copying")
	flag.Parse()
}

func copyObject(bucket, filename string, checkExistence bool, oldProfileConfig, newProfileConfig ProfileConfig) (err error) {
	// Create separate sessions for old and new profiles
	oldSess, err := session.NewSessionWithOptions(session.Options{
		Profile: "old",
		Config: aws.Config{
			Region:      aws.String(oldProfileConfig.Region),
			Endpoint:    aws.String(oldProfileConfig.Endpoint),
			Credentials: credentials.NewStaticCredentials(oldProfileConfig.AccessKey, oldProfileConfig.SecretKey, ""),
			S3ForcePathStyle: aws.Bool(true),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create session for old profile: %w", err)
	}

	newSess, err := session.NewSessionWithOptions(session.Options{
	  Profile: "new",
		Config: aws.Config{
			Region:      aws.String(newProfileConfig.Region),
			Endpoint:    aws.String(newProfileConfig.Endpoint),
			Credentials: credentials.NewStaticCredentials(newProfileConfig.AccessKey, newProfileConfig.SecretKey, ""),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create session for new profile: %w", err)
	}

	// Create S3 clients for both profiles
	oldS3Client := s3.New(oldSess)
	newS3Client := s3.New(newSess)

	if checkExistence {
		// Perform a HEAD request to check if the file exists in the new bucket
		_, err := newS3Client.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(filename),
		})
		if err == nil { // If err is nil, the object exists
			fmt.Printf("[%s] already exists in the destination bucket. Skipping copy.\n", filename)
			return nil // Skip copying since the file exists
		}
		// If the file doesn't exist, proceed with copying (error handling for non-existence not shown for brevity)
	}

        // Start timing just before getting the object
        startTime := time.Now()

	// Download object from old profile bucket
	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	}
	oldObject, err := oldS3Client.GetObject(getObjectInput)
	if err != nil {
		return fmt.Errorf("[%s] failed to get object from old profile: %w", filename, err)
	}
	defer oldObject.Body.Close()

	// Read object body
	objectBytes, err := ioutil.ReadAll(oldObject.Body)
	if err != nil {
		return fmt.Errorf("failed to read object body: %w", err)
	}


	// Initialize the PutObjectInput struct
	putObjectInput := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		Body:   bytes.NewReader(objectBytes),
		ContentType: oldObject.ContentType, // Default to old object's ContentType
	}

	// Determine the Content-Type based on the file extension and set it if applicable
	if strings.HasSuffix(filename, ".jpg") || strings.HasSuffix(filename, ".jpeg") {
		putObjectInput.ContentType = aws.String("image/jpeg")
	} else if strings.HasSuffix(filename, ".png") {
		putObjectInput.ContentType = aws.String("image/png")
	}
	// No else case needed; AWS will handle default Content-Type

	// Upload object to new profile bucket
	newObject, err := newS3Client.PutObject(putObjectInput)
	if err != nil {
		return fmt.Errorf("[%s] failed to put object to new profile: %w", filename, err)
	}

	if *oldObject.ETag != *newObject.ETag {
		return fmt.Errorf("[%s] ETags don't match after copy: %s != %s", filename, *oldObject.ETag, *newObject.ETag)
	}
        // Calculate the elapsed time
        elapsedTime := time.Since(startTime)

        fmt.Printf("[%s] Successfully copied object from old profile to new profile in: %v ms.\n", filename, elapsedTime.Milliseconds())

	return nil
}


func main() {
	if configPath == "" || filename == "" || bucket == "" {
		fmt.Println("Usage: --config <config_path> --filename <file_path> --bucket <bucket_name> [--check]")
		os.Exit(1)
	}

	// Read the configuration
	config, err := ReadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading configuration: %s\n", err)
		os.Exit(1)
	}

	// Number of concurrent workers (configurable)
	numWorkers := 100 // Adjust this value as needed

	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Create buffered scanner
	scanner := bufio.NewScanner(file)

	// Channel to buffer lines for processing
	lineChan := make(chan string, numWorkers)

	// Wait group to track goroutine completion
	var wg sync.WaitGroup

	// Start goroutines for processing
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range lineChan {
				// err := copyObject(bucket, line, checkExistence)
				err := copyObject(bucket, line, checkExistence, config.Profiles.OldProfile, config.Profiles.NewProfile)
				if err != nil {
					fmt.Println("Error processing line:", err)
				}
			}
		}()
	}

	// Go routine to read file line by line and push to channel
	go func() {
		defer close(lineChan)
		for scanner.Scan() {
			lineChan <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("Error reading file:", err)
		}
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	fmt.Println("Processing complete!")
}
