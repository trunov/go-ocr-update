package ocr

import (
	"bytes"
	"image"
	"image/png"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
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
	img, err := renderPageFromBuffer(pdfData, 1)
	if err != nil {
		return "", err
	}

	rgba, ok := img.(*image.RGBA)
	if !ok {
		return "", err
	}

	b, err := encodePNGToBytes(rgba)
	if err != nil {
		return "", err
	}

	client.SetImageFromBytes(b)
	return client.Text()
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

func encodePNGToBytes(img *image.RGBA) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
