# 图片加水印工具 (Image Watermarking Tool)

这是一个 Go 语言编写的命令行工具，用于批量为图片添加包含日期和天数差的水印。

## 功能

*   读取指定输入目录下的图片文件（支持 `.jpg`, `.jpeg`, `.png` 格式）。
*   获取每张图片的创建时间（优先）或修改时间（备选）。
*   计算图片时间戳与预设目标日期之间的天数差。
*   将包含 "日期 时间 (天数差)" 格式的水印添加到图片的左下角。
*   将处理后的图片统一以 PNG 格式保存到指定的输出目录。

## 配置

在 <mcfile name="main.go" path="/Users/hankangkang/go/src/test/photos/main.go"></mcfile> 文件开头的 `const` 和 `var` 部分可以修改以下配置：

*   `inputDir`: (常量) 输入图片所在的目录路径。**请确保此目录存在且包含图片文件。**
*   `outputDir`: (常量) 保存带水印图片的目录路径。如果目录不存在，程序会自动创建。
*   `targetDateStr`: (常量) 目标日期，用于计算天数差，格式必须是 "YYYY-MM-DD"。
*   `fontSize`: (常量) 水印文字的大小（单位：点）。
*   `watermarkOffsetX`: (常量) 水印距离图片左边缘的像素数。
*   `watermarkOffsetY`: (常量) 水印距离图片底边缘的像素数（注意：这是基线到底部的距离）。
*   `watermarkColor`: (变量) 水印文字的颜色，使用 `color.RGBA` 定义。

## 依赖

*   Go 标准库
*   `golang.org/x/image/font`
*   `golang.org/x/image/font/gofont/goregular` (内置 Go 字体)
*   `golang.org/x/image/font/opentype`
*   `golang.org/x/image/math/fixed`

## 构建与运行

1.  **安装 Go 环境:** 确保你的系统已经安装了 Go 开发环境 (版本 >= 1.16)。
2.  **获取代码:** 将 <mcfile name="main.go" path="/Users/hankangkang/go/src/test/photos/main.go"></mcfile> 文件保存在你的工作目录中（例如 `photos` 目录）。
3.  **配置:** 根据需要修改 <mcfile name="main.go" path="/Users/hankangkang/go/src/test/photos/main.go"></mcfile> 文件中的配置常量。
4.  **构建:** 在包含 <mcfile name="main.go" path="/Users/hankangkang/go/src/test/photos/main.go"></mcfile> 的目录下打开终端，运行以下命令编译程序：
    ```bash
    go build
    ```
    这会生成一个名为 `photos` (或在 Windows 上是 `photos.exe`) 的可执行文件。
5.  **运行:** 执行生成的文件：
    ```bash
    ./photos
    ```
    程序会开始处理 `inputDir` 中的图片，并将结果保存到 `outputDir`。处理过程和结果会打印在终端上。

## 代码逻辑

1.  **`main` 函数:**
    *   解析 `targetDateStr` 字符串为 `time.Time` 对象 (`targetDate`)。
    *   加载内置的 Go Regular 字体 (`goregular.TTF`)，并使用 `opentype` 包创建指定大小 (`fontSize`) 的字体 Face (`loadedFace`)。
    *   检查 `inputDir` 是否存在，如果不存在则报错退出。
    *   创建 `outputDir` (如果需要)。
    *   打印配置信息。
    *   读取 `inputDir` 目录内容。
    *   遍历目录中的文件：
        *   跳过子目录。
        *   检查文件扩展名是否为 `.jpg`, `.jpeg`, 或 `.png`。
        *   如果是支持的图片格式，调用 <mcsymbol name="processImage" filename="main.go" path="/Users/hankangkang/go/src/test/photos/main.go" startline="55" type="function"></mcsymbol> 函数处理该图片。
        *   记录处理的图片数量。
    *   输出最终的处理统计信息。

2.  **`processImage` 函数 (处理单张图片):**
    *   打开图片文件。
    *   获取文件信息 (`os.Stat`)。
    *   **获取时间戳:**
        *   尝试通过 `fileInfo.Sys().(*syscall.Stat_t)` 获取底层的系统文件信息。
        *   在 macOS/Unix 上，尝试读取 `Birthtimespec` (创建时间)。
        *   如果成功获取到创建时间，则使用创建时间 (`creationTime`)。
        *   如果无法获取创建时间（例如在非 Unix 系统或获取失败），则回退使用文件的修改时间 (`fileInfo.ModTime()`)，并打印警告信息。
    *   调用 <mcsymbol name="calculateDaysDifference" filename="main.go" path="/Users/hankangkang/go/src/test/photos/main.go" startline="40" type="function"></mcsymbol> 计算获取到的时间戳与 `targetDate` 之间的天数差。
    *   格式化水印字符串，包含日期、时间和天数差，例如 "2023-10-26 10:30:00 (368)"。
    *   使用 `image.Decode` 自动解码图片（支持 JPEG 和 PNG）。需要先 `file.Seek(0, 0)` 重置文件指针。
    *   创建一个新的 `image.RGBA` 画布，大小与原图相同。
    *   将原图绘制到新的 RGBA 画布上。
    *   创建 `font.Drawer`，设置目标画布、水印颜色 (`watermarkColor`) 和加载的字体 Face (`loadedFace`)。
    *   计算水印的绘制坐标 (`Dot`)，X 坐标基于 `watermarkOffsetX`，Y 坐标基于图片高度和 `watermarkOffsetY`。
    *   使用 `d.DrawString` 将水印文本绘制到画布上。
    *   创建输出文件路径。
    *   创建输出文件。
    *   使用 `png.Encode` 将带水印的 RGBA 图像编码为 PNG 格式并写入输出文件。
    *   打印成功处理的信息。

3.  **`calculateDaysDifference` 函数:**
    *   接收两个 `time.Time` 对象 `t1` 和 `t2`。
    *   将两个时间都转换为 UTC 时区，并截断到当天的零点，以忽略时分秒的影响。
    *   计算两个零点时间点之间的 `time.Duration` 差值。
    *   将 `Duration` 转换为小时数，再除以 24，取整数部分得到天数差。

## 注意事项

*   **时间戳获取:** 获取文件创建时间的功能依赖于操作系统。代码中使用了 `syscall` 包，主要针对 macOS/Unix 系统。在 Windows 或其他系统上，可能无法获取到创建时间，程序会自动使用修改时间作为替代。
*   **输出格式:** 所有处理后的图片都会被保存为 PNG 格式，即使原始图片是 JPEG 格式。
*   **错误处理:** 程序包含基本的错误处理和日志记录。如果文件无法打开、解码或保存，会打印错误信息并跳过该文件。关键的初始化错误（如无法解析日期、加载字体、访问目录）会导致程序终止。
