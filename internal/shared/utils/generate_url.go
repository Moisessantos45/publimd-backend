package utils

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

func GenerateURL(path string, params string) string {
	isProduction := os.Getenv("GO_ENV")
	var host = os.Getenv("HOST_API_PROD")
	if isProduction == "dev" {
		host = os.Getenv("HOST_API_DEV")
	}

	if isProduction == "dev" {
		ip := GetOutboundIP()
		host = strings.ReplaceAll(host, "localhost", ip)
	}

	if strings.TrimSpace(params) == "" {
		return fmt.Sprintf("%s/%s", host, path)
	}

	return fmt.Sprintf("%s/%s/%s/%s", host, "api/v1", path, params)
}

func ReplaceHost(host string) (string, error) {
	if strings.TrimSpace(host) == "" {
		return "", nil
	}

	isProduction := os.Getenv("GO_ENV")

	if isProduction != "dev" {
		return host, nil
	}

	u, err := url.Parse(host)
	if err != nil {
		return "", err
	}

	if p := u.Port(); p != "" {
		u.Host = net.JoinHostPort("localhost", p)
	} else {
		u.Host = "localhost"
	}

	return u.String(), nil
	//http: //192.168.100.254:4101/api/v1/storage/file/1774472135848537271
}
