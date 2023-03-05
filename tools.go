package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	m "math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"
const NUM_NONALPHA = 12

// trand is a local wrapper of Rand used to generate random numbers out of package math/rand.
var trand *m.Rand

// init sets up the random generator with a unique Source for this instance of the module.
func init() {
	trand = m.New(m.NewSource(time.Now().Unix()))
}

// Tools is the type used to instantiate this module.
type Tools struct {
	MaxFileSize      int
	AllowedFileTypes []string
}

// UploadedFile contains meta data about a file that was uploaded before.
type UploadedFile struct {
	NewFileName      string
	OriginalFileName string
	FileSize         int64
}

// RandomStringWithAlpha returns a string of size length consisting of random characters. The string
// doesn't start with a non-alphabetic character.
func (t *Tools) RandomStringWithAlpha(length int) string {
	charPool := []rune(randomStringSource)
	result := make([]rune, length)
	for i := range result {
		num := len(charPool) // num = all chars from charPool
		if i == 0 {
			num -= NUM_NONALPHA // num = only alphabetic chars from charPool for first character
		}
		result[i] = charPool[trand.Intn(num)]
	}
	return string(result)
}

// RandomString returns a string of size length consisting of random characters.
func (t *Tools) RandomString(length int) string {
	s, r := make([]rune, length), []rune(randomStringSource)
	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}
	return string(s)
}

func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		// Optional rename parameter was provided - take option from there
		renameFile = rename[0]
	}

	files, err := t.UploadFiles(r, uploadDir, renameFile)
	if err != nil {
		return nil, err
	}
	return files[0], nil
}

func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	renameFile := true
	if len(rename) > 0 {
		// Optional rename parameter was provided - take option from there
		renameFile = rename[0]
	}

	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024 // 1GByte default size
	}

	err := t.CreateDirIfNotExist(uploadDir)
	if err != nil {
		return nil, err
	}

	err = r.ParseMultipartForm(int64(t.MaxFileSize))
	if err != nil {
		return nil, errors.New("uploaded file is too big")
	}

	var uploadedFiles []*UploadedFile

	for _, fHeaders := range r.MultipartForm.File {
		for _, hdr := range fHeaders {
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {
				var uploadedFile UploadedFile
				infile, err := hdr.Open()
				if err != nil {
					return nil, err
				}
				defer infile.Close()

				// Read the first 512 bytes into a buffer to inspect mime type of the file
				buf := make([]byte, 512)
				_, err = infile.Read(buf)
				if err != nil {
					return nil, err
				}

				allowed := false
				fileType := http.DetectContentType(buf)

				if len(t.AllowedFileTypes) > 0 {
					for _, ft := range t.AllowedFileTypes {
						if strings.EqualFold(fileType, ft) {
							allowed = true
						}
					}
				} else {
					allowed = true
				}

				if !allowed {
					return nil, errors.New("uploaded file type is not permitted")
				}

				// Since 512 bytes have been inspected, to upload the file, we need to start from the begin
				_, err = infile.Seek(0, 0)
				if err != nil {
					return nil, err
				}

				uploadedFile.OriginalFileName = hdr.Filename
				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(hdr.Filename))
				} else {
					uploadedFile.NewFileName = hdr.Filename
				}

				// Upload to server
				var outfile *os.File
				defer outfile.Close()
				if outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return nil, err
				} else {
					fileSize, err := io.Copy(outfile, infile)
					if err != nil {
						return nil, err
					}
					uploadedFile.FileSize = fileSize
				}

				uploadedFiles = append(uploadedFiles, &uploadedFile)

				return uploadedFiles, nil

			}(uploadedFiles)
			if err != nil {
				return uploadedFiles, err
			}
		}
	}
	return uploadedFiles, nil
}

func (t *Tools) CreateDirIfNotExist(dir string) error {
	fi, err := os.Stat(dir)
	if os.IsNotExist(err) {
		// Create directory
		err := os.MkdirAll(dir, 0755)
		return err
	}
	if !fi.IsDir() {
		// File does already exist but is not a directory
		return errors.New("file is not a directory")
	}

	return nil
}
