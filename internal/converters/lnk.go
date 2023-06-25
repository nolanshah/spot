package converters

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

type ShortcutHeader struct {
	HeaderSize     uint32
	LinkCLSID      [16]byte
	LinkFlags      uint32
	FileAttributes uint32
	CreationTime   uint64
	AccessTime     uint64
	WriteTime      uint64
	FileSize       uint32
	IconIndex      uint32
	ShowCommand    uint32
	HotKey         uint16
	Reserved1      uint16
	Reserved2      uint32
	Reserved3      uint32
}

func ExtractLinkFromShortcut(inputDir string, inputFileRelPath string) (string, error) {
	inputDirAbs, err := filepath.Abs(inputDir)
	if err != nil {
		log.Error().Err(err).Str("inputDirAbs", inputDir).Msg("Failed to get input absolute path")
		return "", err
	}
	inputFileAbsPath := filepath.Join(inputDirAbs, inputFileRelPath)

	// Open the shortcut file
	file, err := os.Open(inputFileAbsPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Read the shortcut header
	var header ShortcutHeader
	err = binary.Read(file, binary.LittleEndian, &header)
	if err != nil {
		return "", err
	}

	// Check if the file is a valid Windows shortcut
	if string(header.LinkCLSID[:]) != "{00021401-0000-0000-C000-000000000046}" {
		return "", errors.New("not a valid Windows shortcut file")
	}

	// Read the remaining data to extract the URL
	remainingDataSize := header.HeaderSize - 76
	remainingData := make([]byte, remainingDataSize)
	_, err = io.ReadFull(file, remainingData)
	if err != nil {
		return "", err
	}

	// Find the URL prefix
	urlPrefix := []byte("URL")
	index := bytesIndex(remainingData, urlPrefix)
	if index == -1 {
		return "", errors.New("no URL found in the shortcut file")
	}

	// Extract the URL
	url := string(remainingData[index+4:])

	return url, nil
}

func bytesIndex(data []byte, substr []byte) int {
	n := len(data)
	m := len(substr)
	for i := 0; i < n-m+1; i++ {
		if bytesEqual(data[i:i+m], substr) {
			return i
		}
	}
	return -1
}

func bytesEqual(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
