package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chai2010/webp"
	"github.com/fsnotify/fsnotify"
)

var processedFiles = "processed_files.txt"

func main() {
	// 定义命令行参数
	inputDir := flag.String("input", "", "输入目录")
	outputDir := flag.String("output", "", "输出目录")
	quality := flag.Int("quality", 90, "WebP 压缩质量，范围从 1-100")
	numWorkers := flag.Int("workers", 4, "并发 worker 数量")

	// 解析命令行参数
	flag.Parse()

	// 检查输入和输出目录是否提供
	if *inputDir == "" || *outputDir == "" {
		log.Fatalf("必须提供输入和输出目录")
	}

	// 创建输出目录
	err := os.MkdirAll(*outputDir, os.ModePerm)
	if err != nil {
		log.Fatalf("无法创建输出目录: %v", err)
	}

	// 创建一个 channel 用于传递文件路径
	fileChan := make(chan string, 100)
	var wg sync.WaitGroup

	// 启动 worker pool
	for i := 0; i < *numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				processFile(path, *inputDir, *outputDir, *quality)
			}
		}()
	}

	// 创建文件监控器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("无法创建文件监控器: %v", err)
	}
	defer watcher.Close()

	// 启动一个 goroutine 监控文件事件
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					fileChan <- event.Name
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("文件监控错误: %v", err)
			}
		}
	}()

	// 添加监控目录
	err = watcher.Add(*inputDir)
	if err != nil {
		log.Fatalf("无法添加监控目录: %v", err)
	}

	// 递归遍历输入目录并处理文件
	err = filepath.Walk(*inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("无法访问路径 %q: %v\n", path, err)
			return err
		}
		if !info.IsDir() {
			fileChan <- path
		}
		return nil
	})
	if err != nil {
		log.Fatalf("遍历输入目录时出错: %v", err)
	}

	// 等待所有 worker 完成
	wg.Wait()

	fmt.Println("图片批量压缩完成")
}

// 处理文件的函数
func processFile(path, inputDir, outputDir string, quality int) {
	fmt.Printf("正在处理: %s\n", path)

	// 生成文件的唯一标识符（哈希值）
	fileHash, err := fileHash(path)
	if err != nil {
		log.Printf("无法计算文件哈希: %v", err)
		return
	}

	// 判断文件是否已处理
	if isFileProcessed(fileHash) {
		fmt.Printf("文件已处理，跳过: %s\n", path)
		return
	}

	// 获取相对路径并创建相应的输出目录
	relPath, err := filepath.Rel(inputDir, path)
	if err != nil {
		log.Printf("无法获取相对路径: %v", err)
		return
	}
	outputPath := filepath.Join(outputDir, relPath)
	outputDirPath := filepath.Dir(outputPath)
	err = os.MkdirAll(outputDirPath, os.ModePerm)
	if err != nil {
		log.Printf("无法创建输出目录: %v", err)
		return
	}

	// 创建输出文件路径 (保存为 .webp 格式)
	outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".webp"

	// 打开图片文件
	file, err := os.Open(path)
	if err != nil {
		log.Printf("无法打开文件: %v", err)
		return
	}
	defer file.Close()

	// 解码图片
	img, _, err := image.Decode(file) // 忽略 format
	if err != nil {
		log.Printf("无法解码图片: %v", err)
		return
	}

	// 创建输出文件
	outFile, err := os.Create(outputPath)
	if err != nil {
		log.Printf("无法创建输出文件: %v", err)
		return
	}
	defer outFile.Close()

	// 保存为 WebP 格式并设置压缩质量
	options := &webp.Options{Lossless: false, Quality: float32(quality)}
	err = webp.Encode(outFile, img, options)
	if err != nil {
		log.Printf("保存 WebP 图片失败: %v", err)
		return
	}

	// 处理完成，记录文件的哈希值
	recordProcessedFile(fileHash)
}

// 计算文件的哈希值
func fileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// 检查文件是否已处理
func isFileProcessed(hash string) bool {
	file, err := os.Open(processedFiles)
	if err != nil {
		// 文件不存在则表示没有处理过
		if os.IsNotExist(err) {
			return false
		}
		log.Printf("无法打开记录文件: %v", err)
		return false
	}
	defer file.Close()

	var line string
	for {
		_, err := fmt.Fscanf(file, "%s\n", &line)
		if err != nil {
			break
		}
		if line == hash {
			return true
		}
	}
	return false
}

// 记录处理过的文件
func recordProcessedFile(hash string) {
	file, err := os.OpenFile(processedFiles, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("无法打开记录文件: %v", err)
		return
	}
	defer file.Close()

	_, err = fmt.Fprintln(file, hash)
	if err != nil {
		log.Printf("无法写入记录文件: %v", err)
	}
}
