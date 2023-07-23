package converters

import (
	"encoding/xml"
	"io/ioutil"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

type Webloc struct {
	XMLName xml.Name `xml:"plist"`
	Dict    struct {
		Key    string `xml:"key"`
		String string `xml:"string"`
	} `xml:"dict"`
}

func ExtractLinkFromWebloc(inputDir string, inputFileRelPath string) (string, error) {
	inputDirAbs, err := filepath.Abs(inputDir)
	if err != nil {
		log.Error().Err(err).Str("inputDirAbs", inputDir).Msg("Failed to get input absolute path.")
		return "", err
	}
	inputFileAbsPath := filepath.Join(inputDirAbs, inputFileRelPath)

	// Read the contents of the webloc file
	data, err := ioutil.ReadFile(inputFileAbsPath)
	if err != nil {
		return "", err
	}

	// Unmarshal the XML data into a Webloc struct
	var webloc Webloc
	err = xml.Unmarshal(data, &webloc)
	if err != nil {
		return "", err
	}

	// Extract and return the URL
	return webloc.Dict.String, nil
}
