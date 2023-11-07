package handler

import (
	"bufio"
	"bytes"
	"fmt"
	"go-ocr/internal/app/htmlextractor"
	"go-ocr/internal/app/ocr"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gorilla/mux"
	"github.com/otiai10/gosseract/v2"
)

type Handler struct {
	OCRClient *gosseract.Client
}

func NewHandler() *Handler {
	client := gosseract.NewClient()

	return &Handler{
		OCRClient: client,
	}
}

func (h *Handler) ExtractText(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File is missing", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	var text string

	switch {
	case strings.HasSuffix(header.Filename, ".pdf"):
		text, err = ocr.ExtractTextFromPDF(buf.Bytes(), h.OCRClient)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case strings.HasSuffix(header.Filename, ".html"):
		text, err = htmlextractor.ExtractTextFromHTML(buf.Bytes())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Unsupported file type", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("python3", "script.py")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd.Start()

	fmt.Fprintln(stdin, text)

	// Read the response from the Python script
	scanner := bufio.NewScanner(stdout)
	var response string
	if scanner.Scan() {
		response = scanner.Text()
	}

	// Close stdin and wait for the Python script to finish
	stdin.Close()
	cmd.Wait()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

func NewRouter(h *Handler) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/extract-text", h.ExtractText).Methods("POST")

	return r
}
