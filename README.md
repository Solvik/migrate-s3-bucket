# migrate-s3-bucket

Efficiently and securely migrate S3 buckets from one source to another with this high-performance Go script. Designed for seamless migration processes, whether you're dealing with billions of items or ensuring minimal downtime for your services, this tool requires setting your source bucket to read-only mode. Additionally, it necessitates a process capable of attempting to fetch data from the new bucket and, if needed, falling back to the old bucket — akin to a CDN's operation. This setup ensures data integrity and availability throughout the migration process.

## Features

- **High Performance**: Optimized for the rapid migration of S3 bucket contents.
- **Massive Scale Capability**: Successfully tested with migrations of over 2 billion items.
- **Flexibility**: Supports migration across different AWS regions and between various S3-compatible storage providers.

## Prerequisites

Before you start, ensure you have the following:
- Go version 1.15 or higher installed.
- AWS CLI configured with the necessary permissions.

## Usage

To use the script, execute the following command, filling in your specific details for `<config-file>`, `<bucket-name>`, and `<listing-file>`:


```
$ go run migrate.go [--check] --config <config-file> --bucket <bucket-name> --filename <listing-file>
```

Options explained:
- `--check`: (Optional) Verify the migration plan without executing any changes.
- `--config <config-file>`: Path to the YAML configuration file with your AWS credentials and bucket details.
- `--bucket <bucket-name>`: The name of the bucket.
- `--filename <listing-file>`: Path to the file listing the objects to migrate.

## Configuration

Your configuration file should follow this structure:

```yaml
profiles:
  oldProfile:
    region: "us-east-1"
    endpoint: "https://s3.eu-west-1.amazonaws.com"
    accessKey: <your-access-key>
    secretKey: <your-secret-key>
  newProfile:
    region: "eu-west-1"
    endpoint: "https://s3-of-another-hosting-provider.com"
    accessKey: <your-access-key>
    secretKey: <your-secret-key>
```

## Error Handling and Logging

This script provides detailed logging of its operations and any encountered errors. For troubleshooting, verify your AWS CLI permissions and network connectivity to the source and destination endpoints.

## Security Considerations

Always secure your AWS credentials. Avoid hardcoding them in the script and use IAM roles and policies for access control.

## Contributing

We welcome contributions! For bugs, feature requests, or submissions, please open an issue or pull request.

## Missing Features (TODO)
* Support Different Destination Bucket Names: Allow specifying a different bucket name for the destination to enhance flexibility.
* Implement Retry Mechanism: To handle failures more gracefully, a retry mechanism will ensure the script can manage intermittent issues automatically.

## License

This project is licensed under the Apache License 2.0. See the LICENSE file for more details.
