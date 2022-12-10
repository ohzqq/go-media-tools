package ffmeta

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"

	"github.com/ohzqq/avtools/chap"
	"golang.org/x/exp/slices"
	"gopkg.in/ini.v1"
)

const ffmetaComment = ";FFMETADATA1\n"

func LoadJson(d []byte) *Meta {
	meta := NewFFmeta()
	err := json.Unmarshal(d, &meta)
	if err != nil {
		log.Fatal(err)
	}

	//if len(meta.Chaps) > 0 {
	//  for _, c := range meta.Chaps {
	//    ch := chap.NewChapter().SetMeta(c)
	//    meta.Chapters.Chapters = append(meta.Chapters.Chapters, ch)
	//  }
	//}

	return meta
}

func LoadIni(input string) *Meta {
	opts := ini.LoadOptions{}
	opts.Insensitive = true
	opts.InsensitiveSections = true
	opts.IgnoreInlineComment = true
	opts.AllowNonUniqueSections = true

	abs, _ := filepath.Abs(input)
	f, err := ini.LoadSources(opts, abs)
	if err != nil {
		log.Fatal(err)
	}

	ffmeta := NewFFmeta()
	ffmeta.Tags = f.Section("").KeysHash()

	if f.HasSection("chapter") {
		sec, _ := f.SectionsByName("chapter")
		for _, chapter := range sec {
			c := Chapter{}
			err := chapter.MapTo(&c)
			if err != nil {
				log.Fatal(err)
			}
			ch := chap.NewChapter().SetMeta(c)
			ffmeta.Chapters.Chapters = append(ffmeta.Chapters.Chapters, ch)
		}
	}

	return ffmeta
}

func (ff Meta) Dump() []byte {
	ini.PrettyFormat = false

	opts := ini.LoadOptions{
		IgnoreInlineComment:    true,
		AllowNonUniqueSections: true,
	}

	ffmeta := ini.Empty(opts)

	for k, v := range ff.Tags {
		_, err := ffmeta.Section("").NewKey(k, v)
		if err != nil {
			log.Fatal(err)
		}
	}

	var buf bytes.Buffer
	_, err := buf.WriteString(ffmetaComment)
	_, err = ffmeta.WriteTo(&buf)
	if err != nil {
		log.Fatal(err)
	}

	_, err = buf.Write(ff.IniChaps())
	if err != nil {
		log.Fatal(err)
	}

	return buf.Bytes()
}

func (ff Meta) Write(wr io.Writer) error {
	_, err := wr.Write(ff.Dump())
	if err != nil {
		return err
	}
	return nil
}

func (ff Meta) Save() error {
	return ff.SaveAs(ff.name)
}

func (ff Meta) SaveAs(name string) error {
	if name == "" && ff.name == "" {
		name = "tmp"
	}

	file, err := os.Create(name + ".ini")
	if err != nil {
		return err
	}
	defer file.Close()

	err = ff.Write(file)
	if err != nil {
		return err
	}

	return nil
}

var metaTags = []string{
	"title",
	"album",
	"artist",
	"composer",
	"date",
	"year",
	"genre",
	"comment",
	"album_artist",
	"track",
	"language",
	"lyrics",
}

func (ff Meta) DumpJson() []byte {
	meta := make(map[string]interface{})
	meta["mimetype"] = mime.TypeByExtension(filepath.Ext(ff.Filename))
	meta["file"] = ff.Filename
	meta["size"] = ff.Meta.Size
	meta["duration"] = ff.Duration().HHMMSS()
	meta["chapters"] = ff.Chapters
	for key, val := range ff.Meta.Tags {
		if slices.Contains(metaTags, key) {
			meta[key] = val
		}
	}
	data, err := json.Marshal(meta)
	if err != nil {
		log.Fatal(err)
	}
	return data
}
