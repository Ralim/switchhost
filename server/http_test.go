package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/ralim/switchhost/library"
	"github.com/ralim/switchhost/settings"
	"github.com/ralim/switchhost/titledb"
)

func maketestServer(t *testing.T) (*Server, *library.Library, string) {
	temp_folder, err := os.MkdirTemp("", "unit_test")
	if err != nil {
		t.Fatal(err)
	}

	settings := settings.NewSettings(path.Join(temp_folder, "settings.json"))
	settings.ServerMOTD = "SwitchRoooooot" // using different one to ensure its honoured
	titledb := titledb.CreateTitlesDB(settings)
	lib := library.NewLibrary(titledb, settings)
	server := NewServer(lib, titledb, settings)
	return server, lib, temp_folder
}

func TestHTTPServerbasics(t *testing.T) {
	server, lib, temp_folder := maketestServer(t)
	defer os.RemoveAll(temp_folder)
	//Now we can fake poke server handlers
	tempBuffer := bytes.NewBuffer([]byte{})

	err := server.generateJSONPayload(tempBuffer, "test", false)
	if err != nil {
		t.Fatal(err)
	}
	response := tempBuffer.String()
	if response != `{"files":[],"directories":null,"success":"SwitchRoooooot","titledb":{}}` {
		t.Errorf("response doesnt match expected >%s<", response)
	}
	//Now insert a game into the library and run tests with content
	file := &library.FileOnDiskRecord{
		Path:    "../testing_files/UnitTest_[05123A0000000000].nsp",
		TitleID: 0x05123A0000000000,
		Version: 0x0,
		Name:    "UnitTest",
	}
	lib.AddFileRecord(file)
	err = server.generateJSONPayload(tempBuffer, "test", false)
	if err != nil {
		t.Fatal(err)
	}
	response = tempBuffer.String()

	if response != `{"files":[],"directories":null,"success":"SwitchRoooooot","titledb":{}}{"files":[{"url":"http://test/vfile/365418291444842496/0/data.bin#UnitTest [05123A0000000000][v0].nsp","size":1,"title":"UnitTest"}],"directories":null,"success":"SwitchRoooooot","titledb":{}}` {
		t.Errorf("response doesnt match expected >%s<", response)
	}

}

func TestHTTPFileServingJSON(t *testing.T) {
	server, _, temp_folder := maketestServer(t)
	defer os.RemoveAll(temp_folder)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/index.json", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.httpHandleJSON)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `{"files":[],"directories":null,"success":"SwitchRoooooot","titledb":{}}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestHTTPFileServingBinary(t *testing.T) {
	server, lib, temp_folder := maketestServer(t)
	defer os.RemoveAll(temp_folder)

	file := &library.FileOnDiskRecord{
		Path:    "../testing_files/UnitTest_[05123A0000000000].nsp",
		TitleID: 0x05123A0000000000,
		Version: 0x0,
		Name:    "UnitTest",
	}
	lib.AddFileRecord(file)

	req, err := http.NewRequest("GET", "vfile/365418291444842496/0/data.bin", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.httpHandlevFile)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expectedLength := 589016
	if rr.Body.Len() != expectedLength {
		t.Errorf("handler returned unexpected body: got %d bytes want %d bytes",
			rr.Body.Len(), expectedLength)
	}
}

func TestHTTPFileServingBinaryRangeBytes(t *testing.T) {
	server, lib, temp_folder := maketestServer(t)
	defer os.RemoveAll(temp_folder)

	file := &library.FileOnDiskRecord{
		Path:    "../testing_files/UnitTest_[05123A0000000000].nsp",
		TitleID: 0x05123A0000000000,
		Version: 0x0,
		Name:    "UnitTest",
	}
	lib.AddFileRecord(file)

	req, err := http.NewRequest("GET", "vfile/365418291444842496/0/data.bin", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Range", "bytes=0-1023")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.httpHandlevFile)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusPartialContent {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusPartialContent)
	}

	// Check the response body is what we expect.
	expectedLength := 1024
	if rr.Body.Len() != expectedLength {
		t.Errorf("handler returned unexpected body: got %d bytes want %d bytes",
			rr.Body.Len(), expectedLength)
	}
}
