package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	panicSeparator = ": "

	tmplPrefix = "([{"
	tmplSuffix = "}])"
	tmplOption = "missingkey=error"
)

type Template struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Data              map[string]interface{} `json:"data,omitempty"`
}

func toJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func newError(message string) (string, error) {
	return "", errors.New(message)
}

func main() {
	filePath := os.Args[1]

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Panic(filePath, panicSeparator, err)
	}

	if err := GenerateManifests(data, os.Stdout); err != nil {
		log.Panic(filePath, panicSeparator, err)
	}
}

func GenerateManifests(data []byte, out io.Writer) error {
	var object Template
	if err := yaml.Unmarshal(data, &object); err != nil {
		return err
	}
	name := object.GetName()

	funcMap := template.FuncMap{
		"toJson": toJSON,
		"error":  newError,
	}
	tmpl, err := template.New(name).
		Delims(tmplPrefix, tmplSuffix).
		Option(tmplOption).
		Funcs(funcMap).
		ParseFS(virtualFS{}, name)
	if err != nil {
		return err
	}

	return tmpl.Execute(out, object.Data)
}
