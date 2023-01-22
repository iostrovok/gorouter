package fileouter

import (
	"os"
)

type FileOuter struct {
	FileName string
	File     *os.File
}

func New(fileName string) (*FileOuter, error) {
	file, err := os.Open(fileName)

	return &FileOuter{
		FileName: fileName,
		File:     file,
	}, err
}

func (fn *FileOuter) Seek(offset int64, whence int) (int64, error) {
	return fn.File.Seek(offset, whence)
}

func (fn *FileOuter) Read(p []byte) (n int, err error) {
	return fn.File.Read(p)
}
