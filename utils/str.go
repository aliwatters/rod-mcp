package utils

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"text/template"
)

func RandomString(length int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	result := make([]rune, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic("crypto/rand failed: " + err.Error())
		}
		result[i] = letters[n.Int64()]
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
