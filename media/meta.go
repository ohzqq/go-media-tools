package media

import (
	"html/template"

	"github.com/ohzqq/avtools/ff"
	"github.com/ohzqq/avtools/meta"
)

func (m Media) DumpIni() []byte {
	return meta.DumpIni(m)
}

func (m Media) DumpCue() []byte {
	return meta.DumpCueSheet(m.Input.Abs, m)
}

func (m *Media) Probe() *Media {
	p := meta.FFProbe(m.Input.Abs)
	m.Media.SetMeta(p)

	if len(m.Media.Streams()) > 0 {
		for _, stream := range m.Media.Streams() {
			s := Stream{}
			for key, val := range stream {
				switch key {
				case "codec_type":
					s.CodecType = val
				case "codec_name":
					s.CodecName = val
				case "index":
					s.Index = val
				case "cover":
					if val == "true" {
						s.IsCover = true
						m.HasCover = true
					}
				}
			}
			m.streams = append(m.streams, s)
		}
	}

	return m
}

func (m Media) DumpFFMeta() ff.Cmd {
	return meta.DumpFFMeta(m.Input.Abs)
}

var tmplFuncs = template.FuncMap{
	"inc": Inc,
}

func Inc(n int) int {
	return n + 1
}

const cueTmpl = `FILE "{{.Input.Name}}" {{.Input.Ext -}}
{{range $idx, $ch := .Chapters}}
TRACK {{inc $idx}} AUDIO
{{- if eq $ch.Title ""}}
  TITLE "Chapter {{inc $idx}}"
{{- else}}
  TITLE "{{$ch.Title}}"
{{- end}}
  INDEX 01 {{$ch.Start.MMSS}}
{{- end -}}`
