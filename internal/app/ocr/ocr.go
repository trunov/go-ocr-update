package ocr

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/textract"
	"github.com/aws/aws-sdk-go-v2/service/textract/types"
	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
	"github.com/klippa-app/go-pdfium/single_threaded"
)

var pdfiumInstance pdfium.Pdfium
var pool pdfium.Pool
var textractClient *textract.Client

func init() {
	pool = single_threaded.Init(single_threaded.Config{})
	var err error
	pdfiumInstance, err = pool.GetInstance(time.Second * 30)
	if err != nil {
		panic(err)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	textractClient = textract.NewFromConfig(cfg)
}

func ExtractTextFromPDF(pdfData []byte) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return "", fmt.Errorf("configuration error, %v", err)
	}
	textractClient := textract.NewFromConfig(cfg)

	doc, err := pdfiumInstance.OpenDocument(&requests.OpenDocument{File: &pdfData})
	if err != nil {
		return "", err
	}
	defer pdfiumInstance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})

	pageCount, err := pdfiumInstance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	if err != nil {
		return "", err
	}

	var combinedText string
	for i := 0; i < pageCount.PageCount; i++ {
		img, err := renderPage(doc, i+1)
		if err != nil {
			return "", err
		}

		b, err := encodePNGToBytes(img)
		if err != nil {
			return "", err
		}

		// Use AWS Textract to detect text from the image bytes
		resp, err := textractClient.DetectDocumentText(context.TODO(), &textract.DetectDocumentTextInput{
			Document: &types.Document{
				Bytes: b,
			},
		})
		if err != nil {
			return "", err
		}

		for _, block := range resp.Blocks {
			if block.BlockType == types.BlockTypeLine || block.BlockType == types.BlockTypeWord {
				if block.Text != nil {
					combinedText += *block.Text + " "
				}
			}
		}
	}

	return combinedText, nil
}

func renderPage(doc *responses.OpenDocument, page int) (image.Image, error) {
	pageRender, err := pdfiumInstance.RenderPageInDPI(&requests.RenderPageInDPI{
		DPI: 300,
		Page: requests.Page{
			ByIndex: &requests.PageByIndex{
				Document: doc.Document,
				Index:    page - 1,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return pageRender.Result.Image, nil
}

func renderPageFromBuffer(pdfBytes []byte, page int) (image.Image, error) {
	doc, err := pdfiumInstance.OpenDocument(&requests.OpenDocument{File: &pdfBytes})
	if err != nil {
		return nil, err
	}

	defer pdfiumInstance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})

	pageRender, err := pdfiumInstance.RenderPageInDPI(&requests.RenderPageInDPI{
		DPI: 200,
		Page: requests.Page{
			ByIndex: &requests.PageByIndex{
				Document: doc.Document,
				Index:    0,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return pageRender.Result.Image, nil
}

func encodePNGToBytes(img image.Image) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
