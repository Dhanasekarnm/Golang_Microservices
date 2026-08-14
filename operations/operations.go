package operations

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func Unzip(src string, dest string) ([]string, error) {

	log.Println("source file:", src, "\n", "dest file:", dest)

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)
		log.Println("f.Name:", f.Name)
		log.Println("fpath:", fpath)
		log.Println(strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)))
		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
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

func Zip(source, target string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
			//log.Println(header.Name, "is a Directory")
		}
		if header.Name != "./" {
			// 5. Create writer for the file header and save content of the file
			headerWriter, err := writer.CreateHeader(header)
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(headerWriter, f)
			if err != nil {
				log.Println(err)
			}
		}
		return err
	})
}

func DecodeBase64toFile(b64 string, path string) error {
	dec, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return errors.New("failed to decode base64 config string")
	}

	f, err := os.Create(path)
	if err != nil {
		return errors.New("failed to save config file")
	}
	defer f.Close()

	if _, err := f.Write(dec); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}

	return nil
}

func UnPackBot(zip_path string) (string, error) {
	//dir_path := zip_path
	dir_path, filename := filepath.Split(zip_path)
	log.Println("dir_path:", dir_path, "filename:", filename)

	fileInfo, err := os.Stat(zip_path)
	if err != nil {
		log.Println(err)
	}
	fileSize := fileInfo.Size()
	log.Printf("During Unpack, Size of the file: %d bytes\n", fileSize)

	//os.RemoveAll(dir_path)
	os.Mkdir(dir_path, 0755)

	filenames, err := Unzip(zip_path, dir_path+strings.TrimSuffix(filename, filepath.Ext(filename)))

	if err != nil {
		log.Println("Error in Unzip:", err, filenames)
		return "", err
	}

	if len(filenames) > 0 {
		return dir_path + strings.TrimSuffix(filename, filepath.Ext(filename)), nil
	} else {
		return "", errors.New("no files unzipped")
	}

}

func SendLog(sreq Sendlogreq, sendlog_url string) Sendlogresp {

	const Red = "\033[31m"
	var sresp Sendlogresp
	sreq_json, _ := json.Marshal(sreq)
	sreq_json_byte := []byte(sreq_json)

	client := &http.Client{}

	req, err := http.NewRequest("POST", sendlog_url, bytes.NewBuffer(sreq_json_byte))
	if err != nil {
		log.Println(Red+"Request Error: ", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		log.Println(Red+"Response error: ", err)
		sresp.Message = fmt.Sprint(err)
		return sresp
	}
	defer response.Body.Close()
	sresp.Message = fmt.Sprint(response.Status)
	return sresp
}
