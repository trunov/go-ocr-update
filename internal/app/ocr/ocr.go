package ocr

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
	"github.com/klippa-app/go-pdfium/single_threaded"
	"github.com/otiai10/gosseract/v2"
)

var instance pdfium.Pdfium
var pool pdfium.Pool

func init() {
	pool = single_threaded.Init(single_threaded.Config{})
	var err error
	instance, err = pool.GetInstance(time.Second * 30)
	if err != nil {
		panic(err)
	}
}

func ExtractTextFromPDF(pdfData []byte, client *gosseract.Client) (string, error) {
	// this is optional
	client.SetPageSegMode(gosseract.PSM_AUTO)

	doc, err := instance.OpenDocument(&requests.OpenDocument{File: &pdfData})
	if err != nil {
		return "", err
	}
	defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})

	pageCount, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
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

		client.SetImageFromBytes(b)
		text, err := client.Text()
		if err != nil {
			fmt.Println("err", err)
			return "", err
		}
		combinedText += text + "\n"
	}

	return combinedText, nil
}

func renderPage(doc *responses.OpenDocument, page int) (image.Image, error) {
	pageRender, err := instance.RenderPageInDPI(&requests.RenderPageInDPI{
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
	doc, err := instance.OpenDocument(&requests.OpenDocument{File: &pdfBytes})
	if err != nil {
		return nil, err
	}

	defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})

	pageRender, err := instance.RenderPageInDPI(&requests.RenderPageInDPI{
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
