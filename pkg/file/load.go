package file

import (
	"fmt"
	"io"
	"os"
)

func Load(path string, data []byte) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()
	if len(data) == 0 {
		data, err = io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("read %v: %w)", path, err)
		}
		return data, nil
	}
	n, err := file.Read(data)
	if err != nil {
		return nil, fmt.Errorf("read %v: %w)", path, err)
	}
	return data[:n], nil
}
