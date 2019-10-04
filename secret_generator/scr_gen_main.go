package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"
)

const (
	secretOpaqueType = "Opaque"
	apiV1            = "v1"
	secretType       = "Secret"
)

type secretData map[string]string
type metaData struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}
type secret struct {
	ApiVersion string     `json:"apiVersion"`
	Kind       string     `json:"kind"`
	Type       string     `json:"type"`
	Metadata   metaData   `json:"metadata"`
	Data       secretData `json:"data"`
}

var (
	jsonRawScrFileName = ""
	secretName         = ""
	jsonScrFileName    = "test.json"
	applyAfterCreate   = false
)
var (
	saveOutput = false
)

func main() {
	parseFlags()
	if !saveOutput {
		defer os.Remove(jsonScrFileName)
	}
	createSecretFromRaw()
	if applyAfterCreate {
		applySecret()
	}
}

func parseFlags() {
	flag.StringVar(&jsonRawScrFileName, "f", "", "Json raw secret file name.")
	flag.StringVar(&secretName, "n", "", "Secret name.")
	flag.StringVar(&jsonScrFileName, "o", "", "Json secret file name.")
	flag.BoolVar(&applyAfterCreate, "a", false, "Json secret file name.")
	flag.Parse()
	if jsonRawScrFileName == "" {
		log.Fatal("You have to specify raw secret file name using flag -f.")
	}
	saveOutput = jsonScrFileName != ""
	if !saveOutput {
		jsonScrFileName = fmt.Sprintf("%d-secret.json", time.Now().Unix())
	}
}

func extractJson(fileName string, scr *secret) {
	f, err := os.Open(fileName)
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()
	jData, err := ioutil.ReadAll(f)
	if err != nil {
		log.Panic(err)
	}
	err = json.Unmarshal(jData, scr)
	if err != nil {
		log.Panic(err)
	}
	// fill empty fields
	tmp := scr
	tmp.Kind = secretType
	if tmp.Type == "" {
		tmp.Type = secretOpaqueType
	}
	if tmp.ApiVersion == "" {
		tmp.ApiVersion = apiV1
	}
	if tmp.Metadata.Name == "" {
		if secretName == "" {
			log.Fatal("You must specify secret name in raw json secret file or using flag -n.")
		}
		tmp.Metadata.Name = secretName
	}
	if tmp.Data == nil {
		log.Panic("Secret data is empty.")
	}
}
func createSecretFromRaw() {
	rawSecret := secret{}
	extractJson(jsonRawScrFileName, &rawSecret)
	dt := rawSecret.Data
	for k, v := range dt {
		dt[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	jsonScrFileData, err := json.MarshalIndent(&rawSecret, "", "    ")
	if err != nil {
		log.Panic(err)
	}
	_ = os.Remove(jsonScrFileName)
	g, err := os.OpenFile(jsonScrFileName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer g.Close()
	_, err = g.Write(jsonScrFileData)
	if err != nil {
		log.Panic(err)
	}
}
func applySecret() {
	fmt.Printf("kubectl apply -f %q\n", jsonScrFileName)
	cmd := exec.Command("kubectl", "apply", "-f", jsonScrFileName)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		log.Panic(fmt.Sprintf("cmd.Run() failed with %s", err.Error()))
	}
}
