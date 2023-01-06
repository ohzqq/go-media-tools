package media

import (
	"bufio"
	"bytes"
	"html/template"
	"log"
	"os"

	"github.com/ohzqq/avtools/ff"
	"github.com/ohzqq/avtools/meta"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"gopkg.in/ini.v1"
)

func (m *Media) LoadIni(name string) *Media {
	file := NewFile(name)
	if IsPlainText(file.Mimetype) {
		contents, err := os.Open(file.Abs)
		if err != nil {
			log.Fatal(err)
		}
		defer contents.Close()

		scanner := bufio.NewScanner(contents)
		line := 0
		for scanner.Scan() {
			if line == 0 && scanner.Text() == meta.FFmetaComment {
				ini := meta.LoadIni(file.Abs)
				m.SetMeta(ini)
				m.FFmeta = file
				break
			} else {
				log.Fatalln("ffmpeg metadata files need to have ';FFMETADATA1' as the first line")
			}
		}
	}
	return m
}

func (m Media) DumpIni() []byte {
	ini.PrettyFormat = false

	opts := ini.LoadOptions{
		IgnoreInlineComment:    true,
		AllowNonUniqueSections: true,
	}

	ffmeta := ini.Empty(opts)

	for k, v := range m.Tags {
		_, err := ffmeta.Section("").NewKey(k, v)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, chapter := range m.Chapters {
		sec, err := ffmeta.NewSection("CHAPTER")
		if err != nil {
			log.Fatal(err)
		}
		sec.NewKey("TIMEBASE", chapter.Timebase().String())
		sec.NewKey("START", chapter.Start().String())
		sec.NewKey("END", chapter.End().String())
		sec.NewKey("title", chapter.Title())
	}

	var buf bytes.Buffer
	_, err := buf.WriteString(meta.FFmetaComment)
	_, err = ffmeta.WriteTo(&buf)
	if err != nil {
		log.Fatal(err)
	}

	//_, err = buf.Write(ff.IniChaps())
	//if err != nil {
	//  log.Fatal(err)
	//}

	return buf.Bytes()
}

func (m *Media) LoadCue(name string) *Media {
	file := NewFile(name)
	if IsPlainText(file.Mimetype) {
		cue := meta.LoadCueSheet(file.Abs)
		m.SetMeta(cue)
	}
	return m
}

func (m Media) DumpCue() []byte {
	var (
		tmpl = template.Must(template.New("cue").Funcs(tmplFuncs).Parse(cueTmpl))
		buf  bytes.Buffer
	)

	err := tmpl.Execute(&buf, m)
	if err != nil {
		log.Fatal(err)
	}

	return buf.Bytes()
}

func (m *Media) Probe() *Media {
	p := meta.FFProbe(m.Input.Abs)
	m.SetMeta(p)

	if len(m.Media.Streams) > 0 {
		for _, stream := range m.Media.Streams {
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
			m.Streams = append(m.Streams, s)
		}
	}

	return m
}

func (m Media) DumpFFMeta() ff.Cmd {
	cmd := ff.New()
	cmd.In(m.Input.Abs, ffmpeg.KwArgs{"y": ""})
	cmd.Output.Pad("").Name("ffmeta-" + m.Input.Name).Ext(".ini")
	cmd.Output.Set("f", "ffmetadata")
	return cmd
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
