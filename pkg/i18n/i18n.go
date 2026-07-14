package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	once sync.Once
	data map[string]string
)

func load() {
	data = make(map[string]string)
	base := "locales"
	// load English by default
	files := []string{"th.json"}
	for _, f := range files {
		path := filepath.Join(base, f)
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		m := map[string]string{}
		if err := json.Unmarshal(b, &m); err != nil {
			continue
		}
		for k, v := range m {
			// only set if not already present (en.json first)
			if _, ok := data[k]; !ok {
				data[k] = v
			}
		}
	}
}

func Translate(key string, args ...interface{}) string {
	once.Do(load)
	if v, ok := data[key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(v, args...)
		}
		return v
	}
	// fallback to key if not found
	if len(args) > 0 {
		return fmt.Sprintf(key, args...)
	}
	return key
}
