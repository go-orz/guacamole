package guacenc

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestVideo(t *testing.T) {
	logger := &DefaultLogger{Quiet: false}

	checkErr := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	now := time.Now()
	client, err := NewRecordingClient("/Users/zz/WebProjects/next-terminal-commercial/data/recording/734d170a-8c75-48cd-a9c3-67cf46ec6139", logger)
	if err != nil {
		panic(err)
	}

	var index = 0
	var finalLastUpdate int64

	err = os.RemoveAll("./imgs")
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll("./imgs", os.ModePerm)
	if err != nil {
		panic(err)
	}

	client.OnSync(func(img image.Image, lastUpdate int64) {
		if lastUpdate == finalLastUpdate {
			return
		}
		finalLastUpdate = lastUpdate
		index++
		println(index)

		pngName := fmt.Sprintf("./imgs/%d.png", index)
		f, err := os.Create(pngName)
		if err != nil {
			checkErr(err)
		}
		defer f.Close()
		if err = png.Encode(f, img); err != nil {
			checkErr(err)
		}
	})

	client.Start()
	println(time.Since(now).String())
}

func TestGenMp4(t *testing.T) {
	// 转换为视频
	command := exec.Command(
		"ffmpeg",
		"-framerate", "10",
		"-s", "1440x796",
		"-i", "./imgs/%d.png",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"o-10fps.mp4",
	)

	if err := command.Run(); err != nil {
		return
	}
}
