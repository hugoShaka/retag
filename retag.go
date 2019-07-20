package main

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

type imageRef struct {
	registry   string
	repository string
	tag        string
}

func main() {
	log.SetLevel(log.DebugLevel)
	source := "registry.localhost/image1:coucou"
	dest := "registry.localhost/image2"
	log.Info("Got issue names")
	_, err := parseImageRef(source)
	if err != nil {
		fatal("Invalid source image ref", err)
	}
	_, err = parseImageRef(dest)
	if err != nil {
		fatal("Invalid destination image ref", err)
	}

}

func fatal(format string, err error) {
	if err != nil {
		log.Fatalf(format, err)
	} else {
		log.Fatal(format)
	}
	os.Exit(1)
}

func parseImageRef(ref string) (imageRef, error) {
	parts := strings.Split(ref, "/")
	if len(parts) < 2 {
		err := fmt.Errorf("Error parsing image ref, no registry found : %s", ref)
		return imageRef{}, err
	}
	registry := parts[0]
	var tag string
	log.Debugf("registry : %s", registry)

	lastPart := strings.Split(parts[len(parts)-1], ":")
	switch len(lastPart) {
	case 1:
		tag = "latest"
		log.Debugf("tag : %s", tag)
	case 2:
		tag = lastPart[1]
		log.Debugf("tag : %s", tag)
	default:
		err := fmt.Errorf("Error parsing image ref, multiple ':' in last part : %s", ref)
		return imageRef{}, err
	}

	repository := strings.Join(append(parts[1:len(parts)-1], lastPart[:1]...), "/")

	return imageRef{registry, repository, tag}, nil
}
