module grafana-backup-s3

go 1.16

require (
	github.com/aws/aws-sdk-go-v2/config v1.1.2
	github.com/aws/aws-sdk-go-v2/credentials v1.1.2
	github.com/aws/aws-sdk-go-v2/service/s3 v1.2.1
	github.com/mitchellh/go-homedir v1.1.0
	gopkg.in/yaml.v2 v2.4.0
)
