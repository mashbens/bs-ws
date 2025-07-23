package generate

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

// GenerateRandomBytes returns securely random bytes
func GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	return bytes, err
}

// GenerateCSRFToken creates base64-url-safe token
func GenerateCSRFToken() (string, error) {
	bytes, err := GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// GenerateAfAcEncDat creates 8-byte hex string
func GenerateAfAcEncDat() (string, error) {
	bytes, err := GenerateRandomBytes(8)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateSzToken creates realistic sz-token style value
func GenerateSzToken() (string, error) {
	part1, err := GenerateRandomBytes(16)
	if err != nil {
		return "", err
	}
	part2, err := GenerateRandomBytes(64)
	if err != nil {
		return "", err
	}
	part3, err := GenerateRandomBytes(12)
	if err != nil {
		return "", err
	}

	b64 := base64.StdEncoding
	return fmt.Sprintf("%s|%s|%s|%02d|%d",
		b64.EncodeToString(part1),
		b64.EncodeToString(part2),
		b64.EncodeToString(part3),
		8, // static or random
		3,
	), nil
}

// GenerateXSapRI creates 20-byte hex token
func GenerateXSapRI() (string, error) {
	bytes, err := GenerateRandomBytes(20)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateXSapSec creates long base64 string
func GenerateXSapSec() (string, error) {
	bytes, err := GenerateRandomBytes(1300) // ~1800+ base64 chars
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// GenerateDNonPtchaSync generates format: base64|digit|base64
func GenerateDNonPtchaSync() (string, error) {
	part1, err := GenerateRandomBytes(8)
	if err != nil {
		return "", err
	}
	part3, err := GenerateRandomBytes(8)
	if err != nil {
		return "", err
	}
	number := rand.Intn(10)
	return fmt.Sprintf("%s|%d|%s",
		base64.StdEncoding.EncodeToString(part1),
		number,
		base64.StdEncoding.EncodeToString(part3),
	), nil
}
func GenerateDateHeader() string {
	return time.Now().UTC().Format(time.RFC1123)
}

// GenerateXRequestID returns [32hex]:[16hex]:[16hex]
func GenerateXRequestID() (string, error) {
	part1, err := GenerateRandomBytes(16)
	if err != nil {
		return "", err
	}
	part2, err := GenerateRandomBytes(8)
	if err != nil {
		return "", err
	}
	part3 := make([]byte, 8) // 8 null-bytes, as example
	return fmt.Sprintf("%s:%s:%s",
		hex.EncodeToString(part1),
		hex.EncodeToString(part2),
		hex.EncodeToString(part3),
	), nil
}

func RandResponseTime() float64 {
	const min = 202.30000000074506
	const max = 277.19999999995343
	return min + rand.Float64()*(max-min)
}

// GenerateAllHeaders returns a map[string]string of all simulated headers
func GenerateAllHeaders() (map[string]string, error) {
	rand.Seed(time.Now().UnixNano())

	csrf, err := GenerateCSRFToken()
	if err != nil {
		return nil, err
	}
	afAcEncDat, err := GenerateAfAcEncDat()
	if err != nil {
		return nil, err
	}
	szToken, err := GenerateSzToken()
	if err != nil {
		return nil, err
	}
	xSapRI, err := GenerateXSapRI()
	if err != nil {
		return nil, err
	}
	xSapSec, err := GenerateXSapSec()
	if err != nil {
		return nil, err
	}
	dNonPtcha, err := GenerateDNonPtchaSync()
	if err != nil {
		return nil, err
	}
	xRequestID, err := GenerateXRequestID()
	if err != nil {
		return nil, err
	}
	date := GenerateDateHeader()

	randTime := RandResponseTime()

	// Reuse szToken for af-ac-enc-sz-token
	headers := map[string]string{
		//url
		"X-CSRFToken":        csrf,
		"af-ac-enc-dat":      afAcEncDat,
		"sz-token":           szToken,
		"x-sap-ri":           xSapRI,
		"x-sap-sec":          xSapSec,
		"af-ac-enc-sz-token": szToken,
		"d-nonptcha-sync":    dNonPtcha,
		"date":               date,
		"x-request-id":       xRequestID,
		"response_time":      strconv.FormatFloat(randTime, 'f', -1, 64), // ← di sini
		// source url
	}

	return headers, nil
}
