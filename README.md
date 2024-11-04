# composeImage
这是一个基于Go的图片转换程序，在提供Go进行直接编译的同时我们还提供了Docker的部署方式，方便你在多平台进行部署。

## Go

```
git clone https://github.com/Aicnal/composeImage.git
go mod download
```

请不要直接使用`Go run...`进行运行，你必须提供输入和输出目录，压缩质量和线程数量
在正式使用之前请先进行编译：
```
go build go build -o composeImage .
```

之后再指定相关目录
```
./composeImage -input /input -output /output -quality 90 -workers 4
```
## Docker

直接进行Docker构建
```
docker build -t image-compressor:latest .
```

```
docker run -v $(pwd)/input:/input -v $(pwd)/output:/output image-compressor:latest
```

或者你可以使用`docker-compose.yaml`进行统一管理

```yaml
version: '3.8'

services:
  image-compressor:
    image: image-compressor:latest
    volumes:
      - ./input:/input
      - ./output:/output
    restart: always
```

## 功能

- [x] 使用GitHub Actions实现了全自动构建Linux下的Docker Images镜像，并且自动上传到Docker Hub
- [x] 当脚本执行的时候不再对已经存在转码过的图片进行处理，会生成一个`processed_files.txt`文件
- [ ] 使用硬件加速