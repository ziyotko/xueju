package handler

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
)

const maxUploadImageDimension = 12000

// inspectImageUpload derives storage metadata from the decoded image instead of
// trusting the multipart filename. WeChat temporary files can have a filename
// extension that does not match their actual JPEG or PNG content.
func inspectImageUpload(data []byte) (extension string, mimeType string, err error) {
	imageConfig, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || imageConfig.Width < 1 || imageConfig.Height < 1 ||
		imageConfig.Width > maxUploadImageDimension || imageConfig.Height > maxUploadImageDimension {
		return "", "", errors.New("invalid image content")
	}

	switch format {
	case "jpeg":
		return ".jpg", "image/jpeg", nil
	case "png":
		return ".png", "image/png", nil
	default:
		return "", "", errors.New("unsupported image format")
	}
}
