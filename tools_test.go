package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
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
