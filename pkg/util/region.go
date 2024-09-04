package util

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func GetCurrentRegion() (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest("GET", "http://metadata.tencentyun.com/latest/meta-data/placement/region", nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	region := string(body)
	if !strings.HasPrefix(region, "ap-") {
		return "", fmt.Errorf("bad region: %s", region)
	}
	return region, nil
}
