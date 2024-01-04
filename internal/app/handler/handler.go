package handler

import (
	"bytes"
	"encoding/json"
	"go-ocr/internal/app/htmlextractor"
	"go-ocr/internal/app/ocr"
	"io"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/gorilla/mux"
	"github.com/otiai10/gosseract/v2"
)

type Handler struct {
	OCRClient   *gosseract.Client
	RestyClient *resty.Client
}

func NewHandler() *Handler {
	client := gosseract.NewClient()
	restyClient := resty.New()

	return &Handler{
		OCRClient:   client,
		RestyClient: restyClient,
	}
}

type InvoiceData struct {
	InvoiceNumber  string         `json:"invoice_number"`
	InvoiceDate    string         `json:"invoice_date"`
	DueDate        string         `json:"due_date"`
	TotalAmount    string         `json:"total_amount"`
	VATAmount      string         `json:"vat_amount"`
	Client         Client         `json:"client"`
	Supplier       Supplier       `json:"supplier"`
	Items          []Item         `json:"items"`
	PaymentDetails PaymentDetails `json:"payment_details"`
}

type Address struct {
	Street   string `json:"street"`
	City     string `json:"city"`
	Postcode string `json:"postcode"`
	Country  string `json:"country"`
}

type Client struct {
	Name      string  `json:"name"`
	VATNumber string  `json:"vat_number"`
	Address   Address `json:"address"`
	Phone     string  `json:"phone"`
	Email     string  `json:"email"`
}

type Supplier struct {
	Name      string  `json:"name"`
	VATNumber string  `json:"vat_number"`
	Address   Address `json:"address"`
	Phone     string  `json:"phone"`
	Email     string  `json:"email"`
}

type Item struct {
	Description string `json:"description"`
	Quantity    string `json:"quantity"`
	UnitPrice   string `json:"unit_price"`
	Total       string `json:"total"`
	VATRate     string `json:"vat_rate"`
}

type PaymentDetails struct {
	BankName  string `json:"bank_name"`
	IBAN      string `json:"iban"`
	SwiftCode string `json:"swift_code"`
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

	response, err := h.RestyClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{"text": text}).
		Post("http://127.0.0.1:5001/format-invoice-info")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if response.StatusCode() != http.StatusOK {
		http.Error(w, response.String(), response.StatusCode())
		return
	}

	var invoiceData InvoiceData
	err = json.Unmarshal(response.Body(), &invoiceData)
	if err != nil {
		http.Error(w, response.String(), response.StatusCode())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// could be improved
	json.NewEncoder(w).Encode(invoiceData)
}

func NewRouter(h *Handler) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/extract-text", h.ExtractText).Methods("POST")

	return r
}
