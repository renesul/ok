package handler

import (
	"github.com/renesul/ok/web"
)

func readTemplate(name string) string {
	data, err := web.Templates.ReadFile("templates/" + name)
	if err != nil {
		return ""
	}
	return string(data)
}
