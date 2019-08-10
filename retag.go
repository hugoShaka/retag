package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
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

const contentType string = "application/vnd.docker.distribution.manifest.v2+json"

func main() {
	log.SetLevel(log.InfoLevel)

	// Parsing stuff.

	insecurePtr := flag.Bool("insecure", false, "use http instead of https")
	userPtr := flag.String("user", "", "specify auth user")
	passwordPtr := flag.String("pass", "", "specify auth password")
	debugPtr := flag.Bool("debug", false, "sets verbosity level to debug")

	flag.Parse()
	if *debugPtr {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugln("insecure:", *insecurePtr)
	args := flag.Args()
	log.Debugln("tail:", args)

	if len(args) != 2 {
		fatal("Please specify only two arguments (src and dest)", nil)
	}

	source := args[0]
	dest := args[1]
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

	var scheme string
	if *insecurePtr {
		scheme = "http://"
	} else {
		scheme = "https://"
	}
	url := fmt.Sprintf("%s%s/v2/", scheme, sourceRef.registry)
	log.Debugf("Base url : %s", url)

	// Checking if registry is alive and we're logged in.

	logged, err := checkRegistry(url)
	if err != nil {
		fatal("Error checking the registry : %s", err)
	}

	var token string

	switch {
	case !logged && (len(*userPtr) != 0):
		token = loginRegistry(url, *userPtr, *passwordPtr, sourceRef, destRef)
	case (!logged && (len(*userPtr) == 0)):
		log.Fatal("Registry needs authentication, please use -user and -password")
	default:
		log.Infof("Registry reachable")
	}

	image := getManifest(url, sourceRef, token)
	log.Info("Manifest retrieved and parsed")
	log.Debugf("Image data : %s", image)

	// Mouting each blob from source to destination.

	for i := 0; i < len(image.layers); i++ {
		mountBlob(url, image.layers[i], sourceRef, destRef, token)
	}

	// Mounting configuration.

	mountBlob(url, image.config, sourceRef, destRef, token)

	// Posting new manifest.

	postManifest(url, destRef, image, token)

	log.Info("Image successfully retagged")
	log.Exit(0)
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

	// Registry is first part, this parser does not support or intend to support
	// non-FullyQualifiedImageRefs.

	registry := parts[0]
	var tag string
	log.Debugf("registry : %s", registry)

	// Getting the tag.
	lastPart := strings.Split(parts[len(parts)-1], ":")
	switch len(lastPart) {
	case 1:
		// We have no tag, defaulting to latest.
		tag = "latest"
		log.Debugf("tag : %s", tag)
	case 2:
		// We have a tag, using it.
		tag = lastPart[1]
		log.Debugf("tag : %s", tag)
	default:
		// More than a tag ? Image ref likely invalid.
		err := fmt.Errorf("Error parsing image ref, multiple ':' in last part : %s", ref)
		return ImageRef{}, err
	}

	repository := strings.Join(append(parts[1:len(parts)-1], lastPart[:1]...), "/")

	return ImageRef{registry, repository, tag}, nil
}

func checkRegistry(url string) (bool, error) {
	// Basically does a GET on /v2/
	// Will perform auth later.
	res, err := http.Get(url)
	if err != nil {
		fatal("Error contacting the registry : %s", err)
	}

	log.Debugf("Registry response : %v", res)
	defer res.Body.Close()

	return res.StatusCode == http.StatusOK, err
}

type token struct {
	Token string `json:"token"`
}

func loginRegistry(url string, user string, password string, sourceRef ImageRef, destRef ImageRef) string {
	// Contacting the registry to get the challenge realm
	res, err := http.Get(url)
	if err != nil {
		fatal("Error contacting the registry : %s", err)
	}

	log.Debugf("Registry response : %v", res)
	authHeader := res.Header.Get("Www-Authenticate")
	log.Debugf("Www-Authenticate : %s", authHeader)

	// Parsing the header WWW-Authenticate
	re := regexp.MustCompile(`Bearer realm="(https://.*)",service="(.*)"`)
	data := re.FindStringSubmatch(authHeader)
	realm := data[1]
	service := data[2]

	// Constructing the scope
	scope := fmt.Sprintf("repository:%s:pull repository:%s:pull,push", sourceRef.repository, destRef.repository)

	// Constructing the query
	client := &http.Client{}
	req, _ := http.NewRequest("GET", realm, nil)
	q := req.URL.Query()
	q.Add("service", service)
	q.Add("scope", scope)
	req.URL.RawQuery = q.Encode()
	log.Debugf("Requesting token query string : %s", req.URL.String())
	req.SetBasicAuth(user, password)

	// Sending the query
	res, err = client.Do(req)
	if err != nil {
		fatal("Error requesting token : %s", err)
	}
	log.Debugf("Token query answer %v", res)

	token := token{}
	json.NewDecoder(res.Body).Decode(&token)

	log.Info("Token obtained")
	log.Debugf("Token : %s", token.Token)

	return token.Token
}

// Manifest : Struct used to unmarshall a v2 manifest.
type Manifest struct {
	SchemaVersion int        `json:"schemaVersion"`
	MediaType     string     `json:"mediaType"`
	Config        BlobInfo   `json:"config"`
	Layers        []BlobInfo `json:"layers"`
}

// BlobInfo : Struct used to unmarshall a v2 manifest blob info.
type BlobInfo struct {
	MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
	Digest    string `json:"digest"`
}

func getManifest(url string, img ImageRef, token string) Image {
	// Retrieves a manifest and put it into the Image struct.
	url = fmt.Sprintf("%s%s/manifests/%s", url, img.repository, img.tag)
	log.Debugf("Getting manifest at URL : %s", url)
	client := &http.Client{}
	req := createRequest("GET", url, nil, token)
	req.Header.Set("Accept", contentType)
	res, err := client.Do(req)
	if err != nil {
		fatal("Error requesting the manifest : %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		fatal("Non 200 status code recieved when requesting manifest", nil)
	}

	// Decoding JSON.

	manifest := Manifest{}
	err = json.NewDecoder(res.Body).Decode(&manifest)
	if err != nil {
		fatal("Error reading JSON : %s", err)
	}
	log.Debugf("Decoded manifest : %v", manifest)

	// Parsing layers to extract only digests.

	layers := make([]string, 0)
	config := manifest.Config.Digest
	for i := 0; i < len(manifest.Layers); i++ {
		layers = append(layers, manifest.Layers[i].Digest)
	}

	return Image{layers, config, manifest}
}

func mountBlob(baseURL string, blob string, src ImageRef, dst ImageRef, token string) {
	// Mounts a blob from the same repository.

	client := &http.Client{}

	// First, checks if blob exists.
	url := fmt.Sprintf("%s%s/blobs/%s", baseURL, dst.repository, blob)
	req := createRequest("HEAD", url, nil, token)

	res, err := client.Do(req)
	if err != nil {
		fatal("Error checking if blob already present : %s", err)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		// The blob already exists, doing nothing.
		log.Infof("Blob %s already exists", blob)
	case http.StatusNotFound:
		// The blob does not exists, needs to be mounted.
		log.Debugf("Blob %s does not exist, mounting", blob)

		url := fmt.Sprintf("%s%s/blobs/uploads/", baseURL, dst.repository)

		req := createRequest("POST", url, nil, token)

		q := req.URL.Query()
		q.Add("from", src.repository)
		q.Add("mount", blob)
		req.URL.RawQuery = q.Encode()

		log.Debugf("Mouting layer query string : %s", req.URL.String())

		res, err = client.Do(req)
		if err != nil {
			fatal("Error requesting the manifest : %s", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusCreated {
			fatal(fmt.Sprintf("Expecting a 201 created on blob %s, got %d", blob, res.StatusCode), nil)
		}
		log.Infof("Blob %s mounted", blob)
	default:
		// We don't know what's going on, aborting.
		fatal(fmt.Sprintf("Unknown status code %d obtained when mounting blob %s", res.StatusCode, blob), nil)
	}

}

func postManifest(baseURL string, dst ImageRef, img Image, token string) {
	// This uploads a manifest to a given repository. It's technically a PUT.
	// Blobs (layers & config) should be present before upload (put or mounted).

	client := &http.Client{}

	url := fmt.Sprintf("%s%s/manifests/%s", baseURL, dst.repository, dst.tag)
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(img.manifest)

	req := createRequest("PUT", url, buf, token)
	req.Header.Set("Content-Type", contentType)

	res, err := client.Do(req)
	if err != nil {
		fatal("Error requesting the manifest : %s", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		fatal(fmt.Sprintf("Error posting manifest, waiting 201, got %d", res.StatusCode), nil)
	}
	log.Info("Manifest uploaded")
}

func createRequest(verb string, url string, buf io.Reader, token string) *http.Request {
	req, err := http.NewRequest(verb, url, buf)
	if err != nil {
		fatal("Error creating request : %s", err)
	}
	if len(token) != 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	return req
}
