package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // 注册 JPEG 解码器
	"image/png"  // 导入 PNG 编码器，并注册解码器
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall" // <-- 添加 syscall 包
	"time"

	"golang.org/x/image/font"
	// "golang.org/x/image/font/basicfont" // 不再需要 basicfont
	"golang.org/x/image/font/gofont/goregular" // 导入 Go 字体数据
	"golang.org/x/image/font/opentype"         // 导入 OpenType 解析器
	"golang.org/x/image/math/fixed"
)

const (
	inputDir         = "/Users/hankangkang/Downloads/photos" // 输入图片目录
	outputDir        = "./output_images"                    // 输出图片目录
	targetDateStr    = "2024-10-03"                         // 目标日期
	fontSize         = 70.0                                 // 水印字体大小 (点)
	watermarkOffsetX = 100                                  // 水印距离左边像素
	watermarkOffsetY = 100                                  // 水印距离底部像素
)

var (
	targetDate     time.Time                                         // 目标日期的时间对象
	watermarkColor = color.RGBA{R: 255, G: 165, B: 0, A: 255} // 水印颜色：橙色
	loadedFace     font.Face                                         // 全局加载的字体 Face
)

// calculateDaysDifference 计算 t1 和 t2 两个日期之间的天数差异 (基于UTC日期)
func calculateDaysDifference(t1, t2 time.Time) int {
	// 将时间戳转换为UTC并截断到当天的开始
	t1Day := time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.UTC)
	t2Day := time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC)

	// 计算持续时间差
	diff := t2Day.Sub(t1Day)

	// 将 Duration 转换为天数 (取整数部分)
	// 注意：Duration.Hours() 返回浮点数
	days := int(diff.Hours() / 24)
	return days
}

// processImage 处理单个图片文件：添加水印并保存
func processImage(filePath string) {
	// 1. 打开图片文件
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("错误：无法打开文件 '%s': %v", filePath, err)
		return
	}
	defer file.Close()

	// 2. 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("错误：无法获取文件信息 '%s': %v", filePath, err)
		return
	}

	// 尝试获取创建时间 (macOS/Unix specific)
	var creationTime time.Time
	stat, ok := fileInfo.Sys().(*syscall.Stat_t) // 获取底层系统信息
	if ok {
		// 在 macOS 上，Birthtimespec 存储创建时间
		// 注意：不同 Unix 系统的字段名可能不同 (e.g., Crtime on some Linux)
		// 将 syscall.Timespec 转换为 time.Time
		creationTime = time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
	}

	// 如果无法获取创建时间 (例如在非 Unix 系统或获取失败)，则回退到修改时间
	var timestampToUse time.Time
	var timeType string
	if !creationTime.IsZero() { // 检查 creationTime 是否有效
		timestampToUse = creationTime
		timeType = "创建"
	} else {
		timestampToUse = fileInfo.ModTime() // 回退到修改时间
		timeType = "修改"
		log.Printf("警告：无法获取文件 '%s' 的创建时间，将使用修改时间。", filepath.Base(filePath))
	}


	// 3. 计算天数差异 (使用获取到的时间)
	daysDiff := calculateDaysDifference(targetDate, timestampToUse)

	// 4. 格式化水印字符串 (使用获取到的时间和类型)
	dateStr := timestampToUse.Format("2006-01-02 15:04:05") // 格式化日期时间
	watermarkText := fmt.Sprintf("%s (%d)", dateStr, daysDiff) // 添加时间类型

	// 5. 解码图片 (自动检测格式 JPEG/PNG)
	// 重置文件读取指针到开头，因为 Stat() 之后可能移动了指针
	_, err = file.Seek(0, 0)
	if err != nil {
		log.Printf("错误：无法重置文件指针 '%s': %v", filePath, err)
		return
	}
	img, format, err := image.Decode(file) // 重新解码
	if err != nil {
		log.Printf("警告：无法自动解码图片 '%s' (格式: %s): %v", filePath, format, err)
		return
	}
	log.Printf("正在处理: %s (格式: %s, 使用 %s 时间)", filepath.Base(filePath), format, timeType)

	// 6. 创建一个新的 RGBA 画布用于绘制
	bounds := img.Bounds()
	// NewRGBA 创建一个具有指定矩形范围的新的 RGBA 图像。
	rgba := image.NewRGBA(bounds)
	// 将原始图像绘制到新的 RGBA 画布上
	// draw.Src 参数表示直接复制源像素
	draw.Draw(rgba, bounds, img, image.Point{}, draw.Src)

	// 7. 添加水印文本 (使用加载的字体和大小)
	d := &font.Drawer{
		Dst:  rgba,                             // 目标图像
		Src:  image.NewUniform(watermarkColor), // 水印颜色
		Face: loadedFace,                       // 使用加载的 Go Regular 字体 Face
		Dot: fixed.Point26_6{
			// X: 使用 watermarkOffsetX
			X: fixed.Int26_6(watermarkOffsetX * 64),
			// Y: 基线位置 = 图片高度 - 底部偏移量
			Y: fixed.Int26_6((bounds.Max.Y - watermarkOffsetY) * 64),
		},
	}
	d.DrawString(watermarkText) // 在指定位置绘制字符串

	// 8. 保存带水印的图片到输出目录
	outputFilePath := filepath.Join(outputDir, filepath.Base(filePath))
	// 为了简单起见，统一保存为 PNG 格式。如果需要保留原格式，需要根据 'format' 变量选择编码器。
	outFile, err := os.Create(outputFilePath)
	if err != nil {
		log.Printf("错误：无法创建输出文件 '%s': %v", outputFilePath, err)
		return
	}
	defer outFile.Close()

	// 使用 png.Encode 将 RGBA 图像编码为 PNG 格式并写入文件
	if err := png.Encode(outFile, rgba); err != nil {
		log.Printf("错误：无法编码并保存图片 '%s': %v", outputFilePath, err)
		return
	}

	log.Printf("成功添加水印并保存到: %s", outputFilePath)
}

func main() {
	var err error
	// 解析目标日期字符串
	targetDate, err = time.Parse("2006-01-02", targetDateStr)
	if err != nil {
		log.Fatalf("严重错误：无法解析目标日期 '%s': %v", targetDateStr, err)
	}

	// 加载字体文件
	fontData := goregular.TTF // 获取内置字体数据
	fontParsed, err := opentype.Parse(fontData)
	if err != nil {
		log.Fatalf("严重错误：无法解析字体数据: %v", err)
	}

	// 创建指定大小的字体 Face
	// 注意：DPI 通常设为 72，这是屏幕显示的常见标准
	loadedFace, err = opentype.NewFace(fontParsed, &opentype.FaceOptions{
		Size:    fontSize, // 使用常量定义的字体大小
		DPI:     72,       // 标准屏幕 DPI
		Hinting: font.HintingFull, // 字体渲染优化
	})
	if err != nil {
		log.Fatalf("严重错误：无法创建字体 Face: %v", err)
	}

	// 检查输入目录是否存在
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		log.Fatalf("严重错误：输入目录 '%s' 不存在。请创建该目录并将图片放入其中。", inputDir)
	}

	// 确保输出目录存在，如果不存在则创建
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("严重错误：无法创建输出目录 '%s': %v", outputDir, err)
	}

	log.Printf("开始处理目录 '%s' 中的图片...", inputDir)
	log.Printf("目标日期: %s", targetDate.Format("2006-01-02"))
	log.Printf("水印将保存到: %s", outputDir)
	log.Printf("水印字体大小: %.1fpt", fontSize) // 打印字体大小
	log.Printf("水印位置: 左 %dpx, 下 %dpx", watermarkOffsetX, watermarkOffsetY) // 打印水印位置

	// 读取输入目录中的所有文件和子目录
	files, err := os.ReadDir(inputDir)
	if err != nil {
		log.Fatalf("严重错误：无法读取输入目录 '%s': %v", inputDir, err)
	}

	processedCount := 0
	// 遍历目录内容
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(inputDir, file.Name())
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			processImage(filePath)
			processedCount++
		} else {
			log.Printf("跳过非支持文件: %s", file.Name())
		}
	}

	if processedCount == 0 {
		log.Println("未在输入目录中找到任何支持的图片文件 (.jpg, .jpeg, .png)。")
	} else {
		log.Printf("处理完成。共处理了 %d 张图片。", processedCount)
	}
}
