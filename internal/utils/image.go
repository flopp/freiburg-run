package utils

import (
	"os"
	"path/filepath"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

var regularFont *truetype.Font = nil
var fontCache map[int64]font.Face = make(map[int64]font.Face)

func getFont(size float64) (font.Face, error) {
	ssize := int64(size)
	if regularFont == nil {
		font, err := truetype.Parse(goregular.TTF)
		if err != nil {
			return nil, err
		}

		regularFont = font
	}

	if f, ok := fontCache[ssize]; ok {
		return f, nil
	}

	f := truetype.NewFace(regularFont, &truetype.Options{Size: size})
	fontCache[ssize] = f
	return f, nil
}

func GenImage(fileName string, title string, sub1 string, sub2 string, backgroundPNG string) error {
	background, err := gg.LoadPNG(backgroundPNG)
	if err != nil {
		return err
	}

	w := 640
	h := 480
	margin := 16

	sizeBig := 70.0
	w1 := 0.0
	h1 := 0.0
	w2 := 0.0
	h2 := 0.0
	w3 := 0.0
	h3 := 0.0

	dc := gg.NewContext(w, h)

	faceBig, err := getFont(sizeBig)
	if err != nil {
		return err
	}

	for sizeBig >= 16.0 {
		dc.SetFontFace(faceBig)
		w1, h1 = dc.MeasureString(title)
		if w1 > float64(w-margin) {
			sizeBig -= 1.0
			faceBig, err = getFont(sizeBig)
			if err != nil {
				return err
			}
		} else {
			break
		}
	}

	sizeSmall := sizeBig * 2.0 / 3.0
	if sizeSmall < 16.0 {
		sizeSmall = 16.0
	}

	faceSmall, err := getFont(sizeSmall)
	if err != nil {
		return err
	}

	for sizeSmall >= 16.0 {
		dc.SetFontFace(faceSmall)
		w2, h2 = dc.MeasureString(sub1)
		w3, h3 = dc.MeasureString(sub2)
		if w2 > float64(w-margin) || w3 > float64(w-margin) {
			sizeSmall -= 1.0
			faceSmall, err = getFont(sizeSmall)
			if err != nil {
				return err
			}
		} else {
			break
		}
	}

	d1 := h2 / 2.0
	d2 := h2 / 3.0
	hr := h1 + d1 + h2 + d2 + h3
	d := (float64(h) - hr) / 2.0

	// #3e8ed0
	dc.SetRGB255(0x3e, 0x8e, 0xd0)
	dc.Clear()
	dc.DrawImage(background, 0, 0)

	// #3082c5
	dc.SetRGB255(0x30, 0x82, 0xc5)
	dc.DrawRectangle(0, d, float64(w), h1+d2)
	dc.Fill()
	dc.SetRGB255(0xff, 0xff, 0xff)

	dc.SetFontFace(faceBig)
	dc.DrawStringAnchored(title, float64(w)/2.0, d+h1/2.0, 0.5, 0.5)
	dc.SetFontFace(faceSmall)
	dc.DrawStringAnchored(sub1, float64(w)/2.0, d+h1+d1+h2/2.0, 0.5, 0.5)
	dc.DrawStringAnchored(sub2, float64(w)/2.0, d+h1+d1+h2+d2+h3/2.0, 0.5, 0.5)
	//dc.DrawStringAnchored("freiburg.run", float64(w-margin), float64(h-margin), 1.0, 0.0)

	err = os.MkdirAll(filepath.Dir(fileName), 0770)
	if err != nil {
		return err
	}

	err = dc.SavePNG(fileName)
	return err
}

func GenImage2(fileName string, title string, sub string, backgroundPNG string) error {
	background, err := gg.LoadPNG(backgroundPNG)
	if err != nil {
		return err
	}

	w := 640
	h := 480
	margin := 16

	sizeBig := 70.0
	w1 := 0.0
	h1 := 0.0
	w2 := 0.0
	h2 := 0.0

	dc := gg.NewContext(w, h)

	faceBig, err := getFont(sizeBig)
	if err != nil {
		return err
	}

	for sizeBig >= 16.0 {
		dc.SetFontFace(faceBig)
		w1, h1 = dc.MeasureString(title)
		if w1 > float64(w-margin) {
			sizeBig -= 1.0
			faceBig, err = getFont(sizeBig)
			if err != nil {
				return err
			}
		} else {
			break
		}
	}

	sizeSmall := sizeBig * 2.0 / 3.0
	if sizeSmall < 16.0 {
		sizeSmall = 16.0
	}

	faceSmall, err := getFont(sizeSmall)
	if err != nil {
		return err
	}

	for sizeSmall >= 16.0 {
		dc.SetFontFace(faceSmall)
		w2, h2 = dc.MeasureString(sub)
		if w2 > float64(w-margin) {
			sizeSmall -= 1.0
			faceSmall, err = getFont(sizeSmall)
			if err != nil {
				return err
			}
		} else {
			break
		}
	}

	d1 := h2 / 2.0
	hr := h1 + d1 + h2
	d := (float64(h) - hr) / 2.0

	// #3e8ed0
	dc.SetRGB255(0x3e, 0x8e, 0xd0)
	dc.Clear()
	dc.DrawImage(background, 0, 0)

	// #3082c5
	dc.SetRGB255(0x30, 0x82, 0xc5)
	dc.DrawRectangle(0, d, float64(w), h1+d1)
	dc.Fill()
	dc.SetRGB255(0xff, 0xff, 0xff)

	dc.SetFontFace(faceBig)
	dc.DrawStringAnchored(title, float64(w)/2.0, d+h1/2.0, 0.5, 0.5)
	dc.SetFontFace(faceSmall)
	dc.DrawStringAnchored(sub, float64(w)/2.0, d+h1+d1+h2/2.0, 0.5, 0.5)
	//dc.DrawStringAnchored("freiburg.run", float64(w-margin), float64(h-margin), 1.0, 0.0)

	err = os.MkdirAll(filepath.Dir(fileName), 0770)
	if err != nil {
		return err
	}

	err = dc.SavePNG(fileName)
	return err
}
