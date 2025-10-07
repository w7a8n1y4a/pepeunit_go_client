package pepeunit_test

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
)

func writeJSON(path string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func tempFilePath(dir, name string) string {
	return filepath.Join(dir, name)
}

func makeJWTWithUUID(uuid string) string {
	head := base64.RawURLEncoding.EncodeToString([]byte("{}"))
	payloadMap := map[string]interface{}{"uuid": uuid}
	payloadBytes, _ := json.Marshal(payloadMap)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return head + "." + payload + ".sig"
}
