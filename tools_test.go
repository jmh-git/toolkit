package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/fs"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestRandomStringStartsWithAlpha(t *testing.T) {
	tools := Tools{}

	n := 10
	s := tools.RandomStringWithAlpha(n)
	if len(s) != n {
		t.Errorf("Invalid string length - want %d, got %d\n", n, len(s))
	}

	allowedFirst := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for _, r := range s {
		if !strings.ContainsRune(allowedFirst, r) {
			t.Errorf("Random string doesn't start w/ alpha - got %v\n", r)
		}
		break
	}
}

func TestRanddomStringLength(t *testing.T) {
	tools := Tools{}

	n := 10
	s := tools.RandomString(n)
	if len(s) != n {
		t.Errorf("Invalid string length - want %d, got %d\n", n, len(s))
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{
		name:          "allowed, no rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    false,
		errorExpected: false,
	},
	{
		name:          "allowed, rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    true,
		errorExpected: false,
	},
	{
		name:          "not allowed",
		allowedTypes:  []string{"image/jpeg"},
		renameFile:    false,
		errorExpected: true,
	},
}

func TestUploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// setup pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			// Create form data field "file"
			part, err := writer.CreateFormFile("file", "./testdata/openshift-1-458162.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/openshift-1-458162.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		// read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		tools := Tools{}
		tools.AllowedFileTypes = e.allowedTypes
		uploadedFiles, err := tools.UploadFiles(request, "./testdata/uploads", e.renameFile)
		t.Logf("Upload Files: %v, %v\n", uploadedFiles, err)

		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			// Test whether file was truly uploaded
			expFile := fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)
			_, err := os.Stat(expFile)
			if os.IsNotExist(err) {
				t.Errorf("%s: File not uploaded, expected %s\n", e.name, uploadedFiles[0].NewFileName)
			}

			// cleanup
			os.Remove(expFile)
		}

		if e.errorExpected && err == nil {
			t.Errorf("%s: Error expected but none received\n", e.name)
		}

		wg.Wait()
	}
}

func TestUploadOneFile(t *testing.T) {
	for _, e := range uploadTests {
		// setup pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)

		go func() {
			defer writer.Close()

			// Create form data field "file"
			part, err := writer.CreateFormFile("file", "./testdata/openshift-1-458162.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/openshift-1-458162.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		// read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		tools := Tools{}
		uploadedFile, err := tools.UploadOneFile(request, "./testdata/uploads", true)
		t.Logf("Upload Files: %v, %v\n", uploadedFile, err)

		if err != nil {
			t.Error(err)
		}

		// Test whether file was truly uploaded
		expFile := fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName)
		_, err = os.Stat(expFile)
		if os.IsNotExist(err) {
			t.Errorf("%s: File not uploaded, expected %s\n", e.name, uploadedFile.NewFileName)
		}

		// cleanup
		os.Remove(expFile)

	}
}

func TestCreateDirIfNotExist_Existing(t *testing.T) {
	tools := Tools{}

	dir := "./testdata/existingUploadDir"
	err := tools.CreateDirIfNotExist(dir)
	if err != nil {
		t.Error(err)
	}

	fi, err := os.Stat(dir)
	if os.IsNotExist(err) {
		t.Errorf("Directory %s has not been created\n", dir)
	}

	if !fi.IsDir() {
		t.Errorf("%s is not a directory\n", dir)
	}

	modeGot := int(fi.Mode().Perm() & fs.ModePerm)
	modeWant := 0755
	if modeGot != modeWant {
		t.Errorf("%s has incorrect filemode - got %o, expected %o\n", dir, modeGot, modeWant)
	}
}

func TestCreateDirIfNotExist_NotExisting(t *testing.T) {
	tools := Tools{}

	dir := "./testdata/notExistingUploadDir"
	err := tools.CreateDirIfNotExist(dir)
	if err != nil {
		t.Error(err)
	}

	fi, err := os.Stat(dir)
	if os.IsNotExist(err) {
		t.Fatalf("Directory %s has not been created\n", dir)
	}

	if !fi.IsDir() {
		t.Errorf("%s is not a directory\n", dir)
	}

	modeGot := int(fi.Mode().Perm() & fs.ModePerm)
	modeWant := 0755
	if modeGot != modeWant {
		t.Errorf("%s has incorrect filemode - got %o, expected %o\n", dir, modeGot, modeWant)
	}

	// cleanup
	os.Remove(dir)
}

func TestCreateDirIfNotExist_ExistingNotDir(t *testing.T) {
	tools := Tools{}

	dir := "./testdata/existingFile"
	err := tools.CreateDirIfNotExist(dir)
	if err == nil {
		t.Error("Expected error but received none")
	}
}

func TestCreateSlug(t *testing.T) {
	tools := Tools{}

	tests := []struct {
		sentence string
		slug     string
		expErr   bool
	}{
		{sentence: "This is the begin of a story", slug: "this-is-the-begin-of-a-story", expErr: false},
		{sentence: "Hello World!", slug: "hello-world", expErr: false},
		{sentence: "", slug: "", expErr: true},
		{sentence: "!+/()%$-", slug: "", expErr: true},
		{sentence: "!-a=?&%§\"'", slug: "a", expErr: false},
	}

	for _, test := range tests {
		got, err := tools.Slugify(test.sentence)
		if err != nil && test.expErr == false {
			t.Errorf("%s - received error %v\n", test.sentence, err)
		}
		if err == nil && test.expErr {
			t.Errorf("%s - expected error but received none", test.sentence)
		}
		if err == nil && got != test.slug {
			t.Errorf("%s - expected %s, got %s\n", test.sentence, test.slug, got)
		}
	}
}

func TestDownloadStaticFile(t *testing.T) {
	responseRecorder := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	var tools Tools
	tools.DownloadStaticFile(responseRecorder, r, "./testdata", "mycat.png", "kitten.png")

	res := responseRecorder.Result()
	defer res.Body.Close()

	got := res.Header["Content-Length"][0]
	want := "79313"
	if got != want {
		t.Errorf("Content length not as expected, got %s\n", got)
	}

	got = res.Header["Content-Disposition"][0]
	want = "attachment; filename=\"kitten.png\""
	if got != want {
		t.Errorf("Invalid Content-Disposition, got %s\n", got)
	}

	_, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Error while reading file\n")
	}
}

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{
		name:          "good JSON",
		json:          `{"foo": "bar"}`,
		errorExpected: false,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "badly formatted JSON",
		json:          `{"foo": }`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "incorrect type",
		json:          `{"foo": 1}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "two JSON files",
		json:          `{"foo": "1"}{"alpha": "beta"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "empty body",
		json:          ``,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "syntax error in JSON",
		json:          `{"foo": 1"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "unkown field",
		json:          `{"fooo": "1"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "allow unkown field in JSON",
		json:          `{"fooo": "1"}`,
		errorExpected: false,
		maxSize:       1024,
		allowUnknown:  true,
	},
	{
		name:          "missing field name",
		json:          `{jack: "1"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  true,
	},
	{
		name:          "file too large",
		json:          `{"foo": "bar"}`,
		errorExpected: true,
		maxSize:       5,
		allowUnknown:  true,
	},
	{
		name:          "not JSON",
		json:          `Heelo world`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  true,
	},
}

func TestReadJSON(t *testing.T) {
	tool := Tools{}

	for _, test := range jsonTests {
		tool.MaxJSONSize = test.maxSize
		tool.AllowUnknownFields = test.allowUnknown
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(test.json)))
		if err != nil {
			t.Error("Error:", err)
		}

		rr := httptest.NewRecorder()

		err = tool.ReadJSON(rr, req, &decodedJSON)
		if test.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none received\n", test.name)
		}

		if !test.errorExpected && err != nil {
			t.Errorf("%s: expected success, but received error %s\n", test.name, err.Error())
		}

		req.Body.Close()
	}
}

func TestWriteJSON(t *testing.T) {
	tool := Tools{}

	// This implements the ResponseWriter w, called by a handler
	rr := httptest.NewRecorder()

	// This is what we like to write
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	// A map[string]string of headers
	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := tool.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("Failed to write JSON: %v\n", err)
	}
}

func TestErrorJSON(t *testing.T) {
	tool := Tools{}

	// This implements the ResponseWriter w, called by a handler. Once used in this test,
	// we expect a body that has the error response in JSON.
	rr := httptest.NewRecorder()
	err := tool.ErrorJSON(rr, errors.New("Some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	payload := JSONResponse{}

	// Create a decoder from the response's body and decode into variable payload.
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error(err)
	}

	if payload.Error != true {
		t.Error("Expected Error attribute being true, got false")
	}

	if payload.Message != "Some error" {
		t.Errorf("Expected Message attribute being \"%s\", got \"%s\"\n", "Some error", payload.Message)
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d\n", http.StatusServiceUnavailable, rr.Code)
	}
}

func BenchmarkRandomStringWithAlphaShort(b *testing.B) {
	tools := Tools{}
	for i := 0; i < b.N; i++ {
		tools.RandomStringWithAlpha(10)
	}
}

func BenchmarkRandomStringWithAlphaLong(b *testing.B) {
	tools := Tools{}
	for i := 0; i < b.N; i++ {
		tools.RandomStringWithAlpha(100)
	}
}

func BenchmarkRandomStringShort(b *testing.B) {
	tools := Tools{}
	for i := 0; i < b.N; i++ {
		tools.RandomString(10)
	}
}

func BenchmarkRandomStringLong(b *testing.B) {
	tools := Tools{}
	for i := 0; i < b.N; i++ {
		tools.RandomString(100)
	}
}
