package tmpl

import (
	"bobingtech/inspect/util/file"
	"bytes"
	"text/template"
)

type Tmpl struct {
	Template     *template.Template
	TmplContent  string
	TmplFilePath string
	TmplData     interface{}
}

func (t *Tmpl) NewTmpl(tmplName string, tmplContent string, tmplData interface{}) (err error) {
	t.Template = template.New(tmplName)
	t.TmplContent = tmplContent
	t.TmplData = tmplData
	_, err = t.Template.Parse(t.TmplContent)
	if err != nil {
		return err
	}
	return nil
}

func (t *Tmpl) NewTmplFromFile(tmplName string, tmplFilePath string, tmplData interface{}) (err error) {
	t.Template = template.New(tmplName)
	t.TmplContent, err = file.ReadFile(tmplFilePath)
	if err != nil {
		return err
	}

	t.TmplData = tmplData
	_, err = t.Template.Parse(t.TmplContent)
	if err != nil {
		return err
	}
	return nil
}

func (t *Tmpl) Render() (result string, err error) {
	var tpl bytes.Buffer
	if err = t.Template.Execute(&tpl, t.TmplData); err != nil {
		return "", err
	}
	result = tpl.String()
	return result, nil
}
