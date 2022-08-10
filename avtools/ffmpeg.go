package avtools

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
)

type ffmpegCmd struct {
	media *Media
	args  cmdArgs
	opts  *Options
	*Args
}

func NewFFmpegCmd(i string) *ffmpegCmd {
	ff := ffmpegCmd{}

	if i != "" {
		ff.media = NewMedia(i)
	}
	return &ff
}

func (cmd *ffmpegCmd) Options(f *Options) *ffmpegCmd {
	cmd.opts = f
	return cmd
}

func (cmd *ffmpegCmd) ShowMeta() {
	cmd.ParseOptions()
	if cover := cmd.opts.CoverFile; cover != "" {
		cmd.media.AddFormat(cover)
	}
	fmt.Printf("%+V\n", cmd.media.ListFormats())
	fmt.Printf("%+V\n", cmd.media.GetFormat("cover"))
}

func (c *ffmpegCmd) getChapters() (Chapters, error) {
	//if len(c.media.json) == 0 {
	//  c.media.JsonMeta().Unmarshal()
	//}

	switch {
	case c.opts.CueFile != "":
		return LoadCueSheet(c.opts.CueFile).Chapters, nil
	case c.opts.MetaFile != "":
		return LoadFFmetadataIni(c.opts.MetaFile).Chapters, nil
	//case c.media.HasChapters():
	//return c.media.Meta().Chapters, nil
	default:
		return nil, fmt.Errorf("There are no chapters!")
	}
}

func (c *ffmpegCmd) Extract() {
	//c.media.JsonMeta().Unmarshal()
	c.ParseOptions()

	switch {
	case c.opts.CueSwitch:
		c.media.FFmetaChapsToCue()
		return
	case c.opts.CoverSwitch:
		c.AudioCodec = "an"
		c.VideoCodec = "copy"
		c.Output = "cover"
		c.Ext = ".jpg"
	case c.opts.MetaSwitch:
		c.AppendMapArg("post", "f", "ffmetadata")
		c.AudioCodec = "none"
		c.VideoCodec = "none"
		c.Output = "ffmeta"
		c.Ext = ".ini"
	}

	cmd := c.ParseArgs()
	cmd.Run()
}

func (cmd *ffmpegCmd) Join(ext string) {
	cmd.ParseOptions()

	tmp, err := os.CreateTemp("", "audiofiles")
	if err != nil {
		log.Fatal(err)
	}

	files := find(ext)
	for _, f := range files {
		if _, err := tmp.WriteString("file '" + f + "'\n"); err != nil {
			log.Fatal(err)
		}
	}

	cmd.AppendMapArg("pre", "f", "concat")
	cmd.AppendMapArg("pre", "safe", "0")
	cmd.Input = tmp.Name()
	cmd.VideoCodec = "vn"
	cmd.Ext = ext

	c := cmd.ParseArgs()
	c.tmpFile = tmp

	c.Run()
}

func (c *ffmpegCmd) Remove() {
	//c.media.JsonMeta().Unmarshal()
	c.ParseOptions()

	if c.opts.ChapSwitch {
		c.AppendMapArg("post", "map_chapters", "-1")
	}

	if c.opts.CoverSwitch {
		c.VideoCodec = "vn"
	}

	if c.opts.MetaSwitch {
		c.AppendMapArg("post", "map_metadata", "-1")
	}

	cmd := c.ParseArgs()
	cmd.Run()
}

func (cmd *ffmpegCmd) Split() error {
	chaps, err := cmd.getChapters()
	if err != nil {
		return err
	}

	m := cmd.media.GetFormat("audio")
	for i, ch := range chaps {
		NewFFmpegCmd(m.Path).Options(cmd.opts).Cut(ch.StartToSeconds(), ch.EndToSeconds(), i)
	}
	return nil
}

func (cmd *ffmpegCmd) Cut(ss, to string, no int) {
	//cmd.media.JsonMeta().Unmarshal()
	cmd.ParseOptions()

	var (
		count = no + 1
		start = ss
		end   = to
	)

	if cmd.opts.ChapNo != 0 {
		chaps, err := cmd.getChapters()
		if err != nil {
			log.Fatal(err)
		}
		ch := chaps[cmd.opts.ChapNo-1]
		count = cmd.opts.ChapNo
		start = ch.StartToSeconds()
		end = ch.EndToSeconds()
	}

	cmd.PreInput = mapArgs{}
	cmd.num = count

	if start != "" {
		cmd.AppendMapArg("pre", "ss", start)
	}

	if end != "" {
		cmd.AppendMapArg("pre", "to", end)
	}

	c := cmd.ParseArgs()
	c.Run()
}

func (cmd *ffmpegCmd) ParseOptions() *ffmpegCmd {
	cmd.Args = Cfg().GetProfile(cmd.opts.Profile)

	if meta := cmd.opts.MetaFile; meta != "" {
		cmd.media.AddFormat(meta)
	}

	if cover := cmd.opts.CoverFile; cover != "" {
		//NewMedia(cmd.opts.CoverFile).IsImage()
	}

	if cue := cmd.opts.CueFile; cue != "" {
		cmd.media.AddFormat(cue)
	}

	if y := cmd.opts.Overwrite; y {
		cmd.Overwrite = y
	}

	if o := cmd.opts.Output; o != "" {
		cmd.Name = o
	}

	if c := cmd.opts.ChapNo; c != 0 {
		cmd.num = c
	}

	return cmd
}

func (cmd *ffmpegCmd) ParseArgs() *Cmd {
	if log := cmd.LogLevel; log != "" {
		cmd.args.Append("-v", log)
	}

	if cmd.Overwrite {
		cmd.args.Append("-y")
	}

	// pre input
	if pre := cmd.PreInput; len(pre) > 0 {
		cmd.args.Append(pre.Split()...)
	}

	// input

	m := cmd.media.GetFormat("audio")
	if cmd.media != nil {
		cmd.args.Append("-i", m.Path)
	}

	if cmd.Input != "" {
		cmd.args.Append("-i", cmd.Input)
	}

	meta := cmd.opts.MetaFile
	if meta != "" {
		cmd.args.Append("-i", meta)
	}

	cover := cmd.opts.CoverFile
	if cover != "" {
		cmd.args.Append("-i", cover)
	}

	//map input
	idx := 0
	if cover != "" || meta != "" {
		cmd.args.Append("-map", strconv.Itoa(idx)+":0")
		idx++
	}

	if cover != "" {
		cmd.args.Append("-map", strconv.Itoa(idx)+":0")
		idx++
	}

	if meta != "" {
		cmd.args.Append("-map_metadata", strconv.Itoa(idx))
		idx++
	}

	// post input
	if post := cmd.PostInput; len(post) > 0 {
		cmd.args.Append(post.Split()...)
	}

	//video codec
	if codec := cmd.VideoCodec; codec != "" {
		switch codec {
		case "":
		case "none", "vn":
			cmd.args.Append("-vn")
		default:
			cmd.args.Append("-c:v", codec)
			//video params
			if params := cmd.VideoParams.Split(); len(params) > 0 {
				cmd.args.Append(params...)
			}

			//video filters
			if filters := cmd.VideoFilters.Join(); len(filters) > 0 {
				cmd.args.Append("-vf", filters)
			}
		}
	}

	//filter complex
	if filters := cmd.FilterComplex.Join(); len(filters) > 0 {
		cmd.args.Append("-vf", filters)
	}

	//audio codec
	if codec := cmd.AudioCodec; codec != "" {
		switch codec {
		case "":
		case "none", "an":
			cmd.args.Append("-an")
		default:
			cmd.args.Append("-c:a", codec)
			//audio params
			if params := cmd.AudioParams.Split(); len(params) > 0 {
				cmd.args.Append(params...)
			}

			//audio filters
			if filters := cmd.AudioFilters.Join(); len(filters) > 0 {
				cmd.args.Append("-af", filters)
			}
		}
	}

	//output
	var (
		name string
		ext  string
	)

	if out := cmd.Output; out != "" {
		name = out
	}

	if p := cmd.Padding; p != "" {
		name = name + fmt.Sprintf(p, cmd.num)
	}

	media := cmd.media.GetFormat("audio")
	switch {
	case cmd.Ext != "":
		ext = cmd.Ext
	default:
		ext = media.Ext
	}
	cmd.args.Append(name + ext)

	return NewCmd(exec.Command("ffmpeg", cmd.args.args...), cmd.opts.Verbose)
}
