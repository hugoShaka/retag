package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ImageRef : The image reference, but parsed. Contains registry, repository, tag.
type ImageRef struct {
	registry   string
	repository string
	tag        string
}

// Image : An image manifest, the layers and the config are parsed.
type Image struct {
	layers   []string
	config   string
	manifest interface{}
}

func main() {
	log.SetLevel(log.DebugLevel)
	source := "reg.localhost/image1:v1"
	dest := "reg.localhost/image2"
	sourceRef, err := parseImageRef(source)
	if err != nil {
		fatal("Invalid source image ref %s", err)
	}
	destRef, err := parseImageRef(dest)
	if err != nil {
		fatal("Invalid destination image ref %s", err)
	}
	if sourceRef.registry != destRef.registry {
		fatal("Images not in the same registry", nil)
	}

	scheme := "http://"
	url := fmt.Sprintf("%s%s/v2/", scheme, sourceRef.registry)
	log.Debugf("Base url : %s", url)
	logged, err := checkRegistry(url)
	if err != nil {
		fatal("Error checking the registry : %s", err)
	}
	if !logged {
		fatal("Login not implemented yet", nil)
	}
	log.Infof("Registry reachable")
	image := getManifest(url, sourceRef)
	log.Debugf("Image data : %s", image)
}

func fatal(format string, err error) {
	if err != nil {
		log.Fatalf(format, err)
	} else {
		log.Fatal(format)
	}
	os.Exit(1)
}

func parseImageRef(ref string) (ImageRef, error) {
	parts := strings.Split(ref, "/")
	if len(parts) < 2 {
		err := fmt.Errorf("Error parsing image ref, no registry found : %s", ref)
		return ImageRef{}, err
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
		return ImageRef{}, err
	}

	repository := strings.Join(append(parts[1:len(parts)-1], lastPart[:1]...), "/")

	return ImageRef{registry, repository, tag}, nil
}

func checkRegistry(url string) (bool, error) {
	res, err := http.Get(url)
	log.Debugf("Registry response : %v", res)
	defer res.Body.Close()

	return res.StatusCode == http.StatusOK, err
}

// Manifest : Struct used to unmarshall a v2 manifest
type Manifest struct {
	SchemaVersion int        `json:"schemaVersion"`
	MediaType     string     `json:"mediaType"`
	Config        BlobInfo   `json:"config"`
	Layers        []BlobInfo `json:"layers"`
}

// BlobInfo : Struct used to unmarshall a v2 manifest blob info
type BlobInfo struct {
	MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
	Digest    string `json:"digest"`
}

func getManifest(url string, img ImageRef) Image {
	url = fmt.Sprintf("%s%s/manifests/%s", url, img.repository, img.tag)
	log.Debugf("Getting manifest at URL : %s", url)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	res, err := client.Do(req)
	if err != nil {
		fatal("Error requesting the manifest : %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		fatal("Non 200 status code recieved when requesting manifest", nil)
	}

	manifest := Manifest{}
	err = json.NewDecoder(res.Body).Decode(&manifest)
	if err != nil {
		fatal("Error reading JSON : %s", err)
	}
	log.Debugf("Decoded manifest : %v", manifest)

	layers := make([]string, 0)
	config := manifest.Config.Digest
	for i := 0; i < len(manifest.Layers); i++ {
		layers = append(layers, manifest.Layers[i].Digest)
	}
	return Image{layers, config, manifest}
}
