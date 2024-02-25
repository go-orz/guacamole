package guacamole

import (
	"log"
	"os"
	"path/filepath"
	"sync"
)

type Recording struct {
	f      *os.File
	buf    chan []byte
	closer chan struct{}
	once   sync.Once
}

func NewRecording(recordingPath string) (*Recording, error) {
	// 判断目录是否存在
	recordingDir := filepath.Dir(recordingPath)
	if _, err := os.Stat(recordingDir); os.IsNotExist(err) {
		// 创建目录
		if err := os.MkdirAll(recordingDir, os.ModePerm); err != nil {
			return nil, err
		}
	}
	// 创建文件
	f, err := os.Create(recordingPath)
	if err != nil {
		return nil, err
	}
	recording := Recording{
		f:      f,
		buf:    make(chan []byte),
		closer: make(chan struct{}),
	}
	return &recording, nil
}

func (r *Recording) Run() {
	defer r.f.Close()
	for {
		select {
		case <-r.closer:
			return
		case p, ok := <-r.buf:
			if !ok {
				return
			}
			if len(p) == 0 {
				continue
			}
			_, err := r.f.Write(p)
			if err != nil {
				log.Printf("guac recording err: %v", err)
				return
			}
		}
	}
}

func (r *Recording) Send(p []byte) {
	r.buf <- p
}

func (r *Recording) Close() {
	r.once.Do(func() {
		close(r.closer)
	})
}
