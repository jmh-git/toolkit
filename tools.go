package toolkit

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	m "math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	MaxFileSize        int
	AllowedFileTypes   []string
	MaxJSONSize        int
	AllowUnknownFields bool
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

// Slugify is a simple function that creates a slug from a string or that returns an error
// if the slug cannot be created due to an invalid input. Characters not in [a-z] or [0-9]
// are replaced by a dash. The slug neither begins nor ends with a dash.
func (t *Tools) Slugify(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty string not permitted")
	}
	re := regexp.MustCompile(`[^a-z\d]+`)
	result := re.ReplaceAllString(strings.ToLower(s), "-")
	result = strings.Trim(result, "-")
	if len(result) == 0 {
		return "", errors.New("after trimming, slug is of zero length")
	}
	return result, nil
}

// DownloadStaticFile triggers the Save As Dialog in the browser to download a file to the local
// disk rather than rendering the file in the browser.
// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Disposition for further details
// about the Content-Disposition header.
func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, path, file, displayName string) {
	filePath := filepath.Join(path, file)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))
	http.ServeFile(w, r, filePath)
}

// JSONResponse represents the type used for sending JSON data in Go.
type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `string:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ReadJSON reads the body of a request and converts it from JSON int data.
func (t *Tools) ReadJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	maxBytes := 1024 * 1024
	if t.MaxJSONSize != 0 {
		maxBytes = t.MaxJSONSize
	}

	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)

	if !t.AllowUnknownFields {
		dec.DisallowUnknownFields()
	}

	err := dec.Decode(data)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field"):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)

		case errors.As(err, &invalidUnmarshalError):
			return fmt.Errorf("error unmarshalling JSON: %s", err.Error())

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("only one JSON value allowed")
	}

	return nil
}

// WriteJSON takes arbitrary data and writes JSON with headers.
func (t *Tools) WriteJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if len(headers) > 0 {
		for k, v := range headers[0] {
			w.Header()[k] = v
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

// ErrorJSON is a helper function that writes an error response in JSON format and optionally sets the status
// code to the status provided by the caller. If no status was given, http.StatusBadRequest (400) is used.
func (t *Tools) ErrorJSON(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	payload := JSONResponse{
		Error:   true,
		Message: err.Error(),
	}

	return t.WriteJSON(w, statusCode, payload)
}

// PushJSONToRemote posts data to uri as JSON and returns the server's response, status code, and error, if any.
// The last parameter client is optional. The default is http.Client.
func (t *Tools) PushJSONToRemote(uri string, data interface{}, client ...*http.Client) (*http.Response, int, error) {
	// create JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, 0, err
	}

	// check for custom http client
	httpClient := &http.Client{}
	if len(client) > 0 {
		httpClient = client[0]
	}

	// build the request and set the header
	request, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, err
	}

	request.Header.Set("Content-Type", "application/json")

	// call the remote uri
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close() // Important to not forget

	// send response back
	return response, response.StatusCode, nil
}
