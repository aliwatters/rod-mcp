package utils

import (
	"bytes"
	"math/rand"
	"text/template"
)

func RandomString(length int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	result := make([]rune, length)
	for i := range result {
		result[i] = letters[rand.Intn(len(letters))]
	}
	return string(result)
}

func ExecuteTemplate(tmplStr string, res any) (string, error) {
	var out bytes.Buffer
	tmpl, err := template.New("tpl").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	err = tmpl.Execute(&out, res)
	return out.String(), err
}
