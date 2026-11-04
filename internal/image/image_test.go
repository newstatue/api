package image

import (
	"os"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
)

func TestConvertAvatar(t *testing.T) {
	if err := vips.Startup(nil); err != nil {
		t.Fatal("Vips 初始化失败:", err)
	}
	defer vips.Shutdown()

	file, err := os.Open("testdata/avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	data, err := ConvertAvatar(file)
	if err != nil {
		t.Fatal(err)
	}

	img, err := vips.NewImageFromBuffer(data)
	if err != nil {
		t.Fatal("转换结果无法解码:", err)
	}
	defer img.Close()

	if img.Format() != vips.ImageTypeWEBP {
		t.Fatalf("期望 WebP，实际为 %v", img.Format())
	}
	if img.Width() != 256 || img.Height() != 256 {
		t.Fatalf("期望 256×256，实际为 %d×%d", img.Width(), img.Height())
	}

	// 保存到临时目录，方便查看转换结果。
	output := "testdata/avatar.webp"
	if err := os.WriteFile(output, data, 0644); err != nil {
		t.Fatal("保存图片失败:", err)
	}
	t.Log("转换结果:", output)
}
