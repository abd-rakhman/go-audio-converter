package apiserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type server struct {
	router     *mux.Router
	logger     *logrus.Logger
	forwardURL string
}

// Constructor of new server
func newServer(forwardURL string) *server {
	s := &server{
		router:     mux.NewRouter(),
		logger:     logrus.New(),
		forwardURL: forwardURL,
	}
	s.configureRouter()
	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *server) configureRouter() {
	s.router.HandleFunc("/", s.welcomeHandler).Methods("GET")
	s.router.HandleFunc("/upload", s.uploadHandler).Methods("POST")
}

func (s *server) welcomeHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to the API Server"))
}

func (s *server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Retrieve the file
	file, handler, err := r.FormFile("audio_file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save the uploaded file to a temporary location
	tempFile, err := os.CreateTemp("", "upload-*.m4a")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name()) // Clean up

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert the file to mp3
	mp3File, err := convertToMP3(tempFile.Name())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(mp3File) // Clean up

	// Retrieve other form values
	bridge := r.FormValue("bridge")
	aiType := r.FormValue("ai_type")

	s.logger.Printf("Uploaded File: %+v\n", handler.Filename)
	s.logger.Printf("File Size: %+v\n", handler.Size)
	s.logger.Printf("MIME Header: %+v\n", handler.Header)
	s.logger.Printf("Bridge: %s, AI Type: %s\n", bridge, aiType)

	// Forward the file to another server
	err = s.forwardFile(mp3File, bridge, aiType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("File uploaded, converted and forwarded successfully"))
}

func convertToMP3(inputFile string) (string, error) {
	outputFile := inputFile + ".mp3"
	cmd := exec.Command("ffmpeg", "-i", inputFile, outputFile)
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return outputFile, nil
}

func (s *server) forwardFile(filename, bridge, aiType string) error {
	// Prepare a buffer to write the file to
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Open the file for reading
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the file field
	part, err := writer.CreateFormFile("audio_file", filepath.Base(filename))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}

	// Write the bridge and ai_type fields
	err = writer.WriteField("bridge", bridge)
	if err != nil {
		return err
	}
	err = writer.WriteField("ai_type", aiType)
	if err != nil {
		return err
	}

	writer.Close()

	// Send the file to the other server
	req, err := http.NewRequest("POST", s.forwardURL, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// log the response body
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	s.logger.Printf("Response Body: %s\n", buf.String())

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to forward file: %s", resp.Status)
	}

	return nil
}

func (s *server) respond(w http.ResponseWriter, _ *http.Request, code int, data interface{}) {
	w.WriteHeader(code)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (s *server) error(w http.ResponseWriter, r *http.Request, code int, err error) {
	s.logger.Errorf("Code: %d, Error: %v", code, err)
	s.respond(w, r, code, map[string]string{"error": err.Error()})
}
