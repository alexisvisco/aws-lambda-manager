package util

import (
	zip_impl "archive/zip"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func CreateZip(dir string) (string, *os.File, error) {
	sum, err := DirToSha256(dir)
	if err != nil {
		return "", nil, err
	}
	file, err := os.Create(os.TempDir() + "/" + "expected-" + sum + ".zip")
	if err != nil {
		return "", nil, err
	}
	err = Zip("repo-test", file)
	if err != nil {
		return "", nil, err
	}

	return sum, file, err
}

func Zip(directory string, writer io.Writer) error {
	zipWriter := zip_impl.NewWriter(writer)

	err := filepath.Walk(directory, func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil || fileInfo.IsDir() {
			return err
		}

		p := strings.SplitN(filePath, "/", 2)
		if len(p) != 2 {
			return errors.New("invalid path structure")
		}
		archivePath := path.Join(filepath.SplitList(p[1])...)

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		info, err := file.Stat()
		if err != nil {
			return err
		}

		defer file.Close()

		if uint32(info.Mode())&0111 != 0 {
			//
			// Create a header for a binary in order to make aws permission to execute the file
			// Ref: https://github.com/aws/aws-lambda-go/blob/master/cmd/build-lambda-zip/main.go#L50
			//
			headerWriter, err := zipWriter.CreateHeader(&zip_impl.FileHeader{
				CreatorVersion: 3 << 8,
				ExternalAttrs:  0777 << 16,
				Name:           archivePath,
				Method:         zip_impl.Deflate,
			})
			if err != nil {
				return err
			}
			_, err = io.Copy(headerWriter, file)
			if err != nil {
				return err
			}
		} else {
			//
			// Just create a regular file into the zip
			//
			zipFileWriter, err := zipWriter.Create(archivePath)
			if err != nil {
				return err
			}
			_, err = io.Copy(zipFileWriter, file)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	return zipWriter.Close()
}
