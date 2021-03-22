# grafana-backup-s3
backup grafana dashboard to s3 or local

Build for my ubuntu
```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build main.go
```

Get it

```
go get -u github.com/woorui/grafana-backup-s3
```

Run it:
```
grafana-backup-s3 -h  
```

Print
```
Usage of grafana-backup-s3:
  -file string
    
        It's a yaml file, format as follow, build it by yourself
        --------
        url: "GRAFANA_REQUEST_URL"
        apiKeys: "GRAFANA_APIKEYS"
        accessKeyId: "S3_ACCESS_KEY"
        secretAccessKey: "S3_ACCESS_SECRET"
        bucket: "S3_BUCKET"
        region: "S3_REGION"      
        prefix: "S3_UPLOAD_FILE_PATH_PREFIX"
        localDir: "LOCAL_PATH_TO_SAVE_COMPRESS_FILE" # default $HOME/grafana-backup/
        --------   
```

Real run it

```
grafana-backup-s3 -file YOUR_YAML_FILE
```