package main

import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
    "github.com/aws/aws-lambda-go/lambda"
    "context"
	"archive/zip"
    "fmt"
    "io"
    "log"
    "os"
    "path/filepath"
	"strings"
	"bytes"
	"mime"
)


const (
	BUILD_BUCKET string = "build-portfolio.kconley.com"
	PORTFOLIO_BUCKET string = "portfolio.kconley.com"
)


type MyEvent struct {
    Name string `json:"name"`
}

func HandleRequest(ctx context.Context, name MyEvent) (string, error) {
    sess, _ := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	downloader := s3manager.NewDownloader(sess)

	f, err := os.Create("package.zip")
	if err != nil {
        log.Fatal(err)
	}
	defer f.Close()

	exists, err := exists("output")
	if !exists {
		err = os.Mkdir("output", 0755)
		if err != nil {
			log.Fatal(err)
		}
	} 

	n, err := downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(BUILD_BUCKET),
		Key:    aws.String("portfoliobuild.zip"),
	})
	if err != nil {
        log.Fatal(err)
	}
	log.Print(n)
	
    files, err := Unzip("package.zip", "output")
    if err != nil {
        log.Fatal(err)
    }

	fmt.Println("Unzipped:\n" + strings.Join(files, "\n"))
	

	for i:=0 ; i < len(files) ; i++ {
		err = AddFileToS3(sess, files[i], PORTFOLIO_BUCKET)
		if err != nil {
			log.Fatal(err)
		}
	}


	// clean up file system
	os.Remove("package.zip")
	os.RemoveAll("output/")

    return "done", err
}


func main() {
	lambda.Start(HandleRequest)
}

// exists returns whether the given file or directory exists
func exists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil { return true, nil }
    if os.IsNotExist(err) { return false, nil }
    return true, err
}


// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileToS3(s *session.Session, fileDir string, S3_BUCKET string) error {
    // Open the file for use
    file, err := os.Open(fileDir)
    if err != nil {
        return err
    }
    defer file.Close()

    // Get file size and read the file content into a buffer
    fileInfo, _ := file.Stat()
    var size int64 = fileInfo.Size()
    buffer := make([]byte, size)
	file.Read(buffer)

	log.Print(mime.TypeByExtension("." + strings.Split(fileDir, ".")[1]))

    // Config settings: this is where you choose the bucket, filename, content-type etc.
    // of the file you're uploading.
    _, err = s3.New(s).PutObject(&s3.PutObjectInput{
        Bucket:               aws.String(S3_BUCKET),
        Key:                  aws.String(strings.Split(fileDir, "output/")[1]),
        ACL:                  aws.String("public-read"),
        Body:                 bytes.NewReader(buffer),
        ContentLength:        aws.Int64(size),
        ContentType:          aws.String(mime.TypeByExtension("." + strings.Split(fileDir, ".")[1])),
	})
	
    return err
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {

    var filenames []string

    r, err := zip.OpenReader(src)
    if err != nil {
        return filenames, err
    }
    defer r.Close()

    for _, f := range r.File {

        // Store filename/path for returning and using later on
        fpath := filepath.Join(dest, f.Name)

        // Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
        if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
            return filenames, fmt.Errorf("%s: illegal file path", fpath)
        }

        filenames = append(filenames, fpath)

        if f.FileInfo().IsDir() {
            // Make Folder
            os.MkdirAll(fpath, os.ModePerm)
            continue
        }

        // Make File
        if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
            return filenames, err
        }

        outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
        if err != nil {
            return filenames, err
        }

        rc, err := f.Open()
        if err != nil {
            return filenames, err
        }

        _, err = io.Copy(outFile, rc)

        // Close the file without defer to close before next iteration of loop
        outFile.Close()
        rc.Close()

        if err != nil {
            return filenames, err
        }
    }
    return filenames, nil
}