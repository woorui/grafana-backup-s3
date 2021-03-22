package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

type Config struct {
	URL             string `yaml:"url"`
	ApiKeys         string `yaml:"apiKeys"`
	AccessKeyId     string `yaml:"accessKeyId"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	Bucket          string `yaml:"bucket"`
	Region          string `yaml:"region"`
	Prefix          string `yaml:"prefix"`
	LocalDir        string `yaml:"localDir"`
}

// Search is grafana dashboard search api result
type Search struct {
	ID          int           `json:"id"`
	UID         string        `json:"uid"`
	Title       string        `json:"title"`
	URI         string        `json:"uri"`
	URL         string        `json:"url"`
	Slug        string        `json:"slug"`
	Type        string        `json:"type"`
	Tags        []interface{} `json:"tags"`
	IsStarred   bool          `json:"isStarred"`
	FolderID    int           `json:"folderId,omitempty"`
	FolderUID   string        `json:"folderUid,omitempty"`
	FolderTitle string        `json:"folderTitle,omitempty"`
	FolderURL   string        `json:"folderUrl,omitempty"`
}

func main() {
	p := flag.String("file", "", `
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
	`)
	flag.Parse()

	c := readConfig(*p)
	Do(c)
}

func readConfig(p string) *Config {
	b, err := ioutil.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatal("It needs a config file by -file flag")
		}
		log.Fatalf("load config file error, %s \n", err.Error())
	}
	c := new(Config)
	if err := yaml.Unmarshal(b, c); err != nil {
		fmt.Printf("load config file error %s \n", err.Error())
	}
	return c
}

func UploadFileToS3(c *Config, src string) error {
	if c.AccessKeyId == "" || c.SecretAccessKey == "" || c.Bucket == "" || c.Region == "" || c.Prefix == "" {
		fmt.Println("s3 configuation incomplete, skip upload")
		return nil
	}
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(c.AccessKeyId, c.SecretAccessKey, ""),
		),
	)
	if err != nil {
		return err
	}
	filename := path.Join(c.Prefix, filepath.Base(src))
	fmt.Println(filename)
	_, err = s3.NewFromConfig(cfg).PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &c.Bucket,
		Key:    &filename,
		Body:   file,
	})
	if err != nil {
		return err
	}
	fmt.Printf("upload success and local at s3://%s/%s \n", c.Bucket, filename)
	return nil
}

func GrafanaHttpGet(c *Config, path string) (b []byte, err error) {
	u, err := url.Parse(c.URL)
	if err != nil {
		return
	}
	u.Path = path
	fmt.Printf("request -> %s \n", u.String())
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.ApiKeys))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func FetchGrafanaSearch(c *Config) ([]Search, error) {
	s := []Search{}
	data, err := GrafanaHttpGet(c, "/api/search")
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	if err != nil {
		return s, err
	}
	return s, nil
}

func FetchGrafanaDashBoard(c *Config, uri string) ([]byte, error) {
	return GrafanaHttpGet(c, fmt.Sprintf("/api/dashboards/%s", uri))
}

func Do(c *Config) {
	if c.LocalDir == "" {
		dir, err := homedir.Dir()
		if err != nil {
			log.Fatalln("get home dir error", err.Error())
		}
		c.LocalDir = path.Join(dir, "grafana-backup")
	}
	root := path.Join(c.LocalDir, strconv.Itoa(int(time.Now().Unix())))
	searchs, err := FetchGrafanaSearch(c)
	if err != nil {
		log.Fatalln("fetch grafana search api failed", err)
	}

	total, failed := 0, 0
	for _, item := range searchs {
		if item.Type == "dash-db" {
			total++
			content, err := FetchGrafanaDashBoard(c, item.URI)
			if err != nil {
				failed++
				fmt.Println("fetch grafana dashboard api failed", err)
				continue
			}
			folderTitle := item.FolderTitle
			if item.FolderTitle == "" {
				folderTitle = "General"
			}
			folder := path.Join(root, folderTitle)
			if _, err := os.Stat(folder); os.IsExist(err) {
				failed++
				fmt.Printf("%s folder is existed next \n", folder)
				continue
			}
			err = os.MkdirAll(folder, os.ModePerm)
			if err != nil {
				failed++
				fmt.Printf("create %s folder error. \n", err)
				continue
			}
			filename := path.Join(folder, item.Title+".json")
			err = ioutil.WriteFile(filename, content, os.ModePerm)
			if err != nil {
				failed++
				fmt.Printf("write %s error %s \n", filename, err.Error())
				continue
			}
		}
	}
	var buf bytes.Buffer
	err = compress(root, &buf)
	if err != nil {
		log.Fatalf("compress %s error %s \n", root, err.Error())
	}
	zipname := fmt.Sprintf("%s.tar.gzip", root)
	zipFile, err := os.OpenFile(zipname, os.O_CREATE|os.O_RDWR, os.FileMode(0600))
	if err != nil {
		log.Fatalf("compress %s error %s \n", root, err.Error())
	}
	defer zipFile.Close()
	if _, err := io.Copy(zipFile, &buf); err != nil {
		log.Fatalf("compress %s error %s \n", root, err.Error())
	}
	if err := os.RemoveAll(root); err != nil {
		fmt.Printf("remove folder %s error %s \n", root, err.Error())
	}

	if err := UploadFileToS3(c, zipname); err != nil {
		fmt.Printf("upload to s3 %s \n", err.Error())
	}

	log.Printf("save grafana dashboard success, here is %s, total %d, failed %d \n", zipname, total, failed)
}

// compress compress a folder using tar and gzip (works with nested folders)
// more see: <https://gist.github.com/mimoo/25fc9716e0f1353791f5908f94d6e726>
func compress(src string, buf io.Writer) error {
	zr := gzip.NewWriter(buf)
	tw := tar.NewWriter(zr)

	filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		header.Name = "dashboards" + diffpath(src, file)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}
		return nil
	})

	if err := tw.Close(); err != nil {
		return err
	}
	if err := zr.Close(); err != nil {
		return err
	}
	return nil
}

// diffpath return string that omit same prefix between a and b,
func diffpath(a, b string) string {
	long, short := b, a
	if len(a) > len(b) {
		long, short = a, b
	}
	if len(a) == len(b) {
		return ""
	}
	i := 0

	for i < len(short) {
		if short[i] != long[i] {
			break
		}
		i++
	}
	return long[i:]
}
