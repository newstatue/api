package image

import (
	"fmt"
	"io"

	"github.com/davidbyttow/govips/v2/vips"
)

func ConvertAvatar(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("读取图片: %w", err)
	}

	img, err := vips.NewThumbnailFromBuffer(
		data, 256, 256, vips.InterestingCentre,
	)
	if err != nil {
		return nil, fmt.Errorf("生成头像: %w", err)
	}
	defer img.Close()

	params := vips.NewWebpExportParams()
	params.Quality = 80
	params.StripMetadata = true

	result, _, err := img.ExportWebp(params)
	if err != nil {
		return nil, fmt.Errorf("编码 WebP: %w", err)
	}

	return result, nil
}
