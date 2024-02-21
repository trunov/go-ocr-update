package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
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

type Invoice struct {
	InvoiceNumber string `json:"invoice_number"`
	InvoiceDate   string `json:"invoice_date"`
	DueDate       string `json:"due_date"`
	TotalAmount   string `json:"total_amount"`
	VatAmount     string `json:"vat_amount"`
	Client        struct {
		Name      string `json:"name"`
		VATNumber string `json:"vat_number"`
		Address   struct {
			Street   string `json:"street"`
			City     string `json:"city"`
			Postcode string `json:"postcode"`
			Country  string `json:"country"`
		} `json:"address"`
		Phone string `json:"phone"`
		Email string `json:"email"`
	} `json:"client"`
	Supplier struct {
		Name      string `json:"name"`
		VATNumber string `json:"vat_number"`
		Address   struct {
			Street   string `json:"street"`
			City     string `json:"city"`
			Postcode string `json:"postcode"`
			Country  string `json:"country"`
		} `json:"address"`
		Phone string `json:"phone"`
		Email string `json:"email"`
	} `json:"supplier"`
	Items []struct {
		Description string `json:"description"`
		Quantity    string `json:"quantity"`
		UnitPrice   string `json:"unit_price"`
		Total       string `json:"total"`
		VatRate     string `json:"vat_rate"`
	} `json:"items"`
	PaymentDetails struct {
		BankName  string `json:"bank_name"`
		IBAN      string `json:"iban"`
		SwiftCode string `json:"swift_code"`
	} `json:"payment_details"`
}

type APIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewHandler() *Handler {
	client := gosseract.NewClient()
	restyClient := resty.New()

	return &Handler{
		OCRClient:   client,
		RestyClient: restyClient,
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
		text, err = ocr.ExtractTextFromPDF(buf.Bytes())
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

	// Define the instruction as a string
	aiInstruction := `Please process the following invoice text and extract the necessary information. Format your response as a JSON object only, exactly as shown in this structure:

{
    "invoice_number": "",
    "invoice_date": "",
    "due_date": "",
    "total_amount": "",
    "vat_amount": "",
    "client": {
        "name": "",
        "vat_number": "",
        "address": {
            "street": "",
            "city": "",
            "postcode": "",
            "country": ""
        },
        "phone": "",
        "email": ""
    },
    "supplier": {
        "name": "",
        "vat_number": "",
        "address": {
            "street": "",
            "city": "",
            "postcode": "",
            "country": ""
        },
        "phone": "",
        "email": ""
    },
    "items": [
        {
            "description": "",
            "quantity": "",
            "unit_price": "",
            "total": "",
            "vat_rate": ""
        }
    ],
    "payment_details": {
        "bank_name": "",
        "iban": "",
        "swift_code": ""
    }
}

Note: Please ensure the response contains only the JSON object with fields filled as applicable based on the invoice text. Do not include any additional messages or comments. If certain information is unavailable in the invoice text, leave the corresponding JSON fields empty or use 'null'. 

Invoice text:\n` + text // Assuming 'text' contains the invoice text

	// Now, use this string in your request
	response, err := h.RestyClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"model": "Mixtral-8x7B-instruct-exl2",
			"messages": []map[string]string{
				{
					"role":    "system",
					"content": "You are a helpful assistant.",
				},
				{
					"role":    "user",
					"content": aiInstruction,
				},
			},
		}).
		Post("http://127.0.0.1:5001/v1/chat/completions")

	// response, err := h.RestyClient.R().
	// 	SetHeader("Content-Type", "application/json").
	// 	SetBody(map[string]string{"text": text}).
	// 	Post("http://127.0.0.1:5001/format-invoice-info")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if response.StatusCode() != http.StatusOK {
		http.Error(w, response.String(), response.StatusCode())
		return
	}

	var apiResponse APIResponse
	if err := json.Unmarshal(response.Body(), &apiResponse); err != nil {
		fmt.Println("JSON Parsing Error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	content := apiResponse.Choices[0].Message.Content

	var invoice Invoice
	if err := json.Unmarshal([]byte(content), &invoice); err != nil {
		fmt.Println("JSON Parsing Error (Content):", err)
		return
	}

	json.NewEncoder(w).Encode(invoice)
}

func NewRouter(h *Handler) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/extract-text", h.ExtractText).Methods("POST")

	return r
}
