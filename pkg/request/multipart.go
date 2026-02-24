package request

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// fileField 文件上传字段
type fileField struct {
	fieldName string
	filePath  string
}

// buildMultipart 构建 multipart/form-data 请求体
// 返回 body reader、Content-Type header、error
func buildMultipart(formData map[string]string, files []fileField) (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 写入表单字段
	for k, v := range formData {
		if err := writer.WriteField(k, v); err != nil {
			return nil, "", fmt.Errorf("%w: %w", ErrMarshal, err)
		}
	}

	// 写入文件字段
	for _, f := range files {
		file, err := os.Open(f.filePath)
		if err != nil {
			return nil, "", fmt.Errorf("%w: %w", ErrMarshal, err)
		}

		err = func() error {
			defer file.Close()

			part, err := writer.CreateFormFile(f.fieldName, filepath.Base(f.filePath))
			if err != nil {
				return err
			}

			_, err = io.Copy(part, file)
			return err
		}()
		if err != nil {
			return nil, "", fmt.Errorf("%w: %w", ErrMarshal, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("%w: %w", ErrMarshal, err)
	}

	return &buf, writer.FormDataContentType(), nil
}
