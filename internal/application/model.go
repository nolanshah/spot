package application

import (
	"html/template"
	"strings"
	"time"
)

type TPage struct {
	SourcePath      string
	TemplatePath    string
	DestinationPath string
	UrlPath         string
	Title           string
	Description     string
	CreatedAt       time.Time
	Tags            []string
	Metadata        map[string]string
}

type TPageList struct {
	List []TPage
}

func (tpl *TPageList) FilterByTag(tag string) (ret []TPage) {
	for _, tp := range tpl.List {
		for _, g := range tp.Tags {
			if g == tag {
				ret = append(ret, tp)
				break
			}
		}
	}
	return
}

func (tpl *TPageList) FilterByUrlPathPrefix(prefix string) (ret []TPage) {
	for _, tp := range tpl.List {
		if strings.HasPrefix(tp.UrlPath, prefix) {
			ret = append(ret, tp)
		}
	}
	return
}

func (tpl *TPageList) FilterByMetadata(key string, val string) (ret []TPage) {
	for _, tp := range tpl.List {
		if v, ok := tp.Metadata[key]; ok && val == v {
			ret = append(ret, tp)
		}
	}
	return
}

type TSite struct {
	Title       string
	Description string
}

type TData struct {
	Page     TPage
	Contents template.HTML
	Pages    TPageList
	Site     TSite
}
