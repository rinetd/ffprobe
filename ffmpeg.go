package ffmpeg

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Info struct {
	Format  map[string]string
	Streams []map[string]string
}

func readSection(r *bufio.Reader, end string) (map[string]string, error) {
	ret := make(map[string]string)
	for {
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		if line == end {
			break
		}
		ss := strings.SplitN(line, "=", 2)
		opt := ss[0]
		val := ss[1]
		if _, ok := ret[opt]; ok {
			return nil, errors.New(fmt.Sprint("duplicate option:", opt))
		}
		ret[opt] = val
	}
	return ret, nil
}

func readLine(r *bufio.Reader) (line string, err error) {
	for {
		var (
			buf []byte
			isP bool
		)
		buf, isP, err = r.ReadLine()
		if err != nil {
			return
		}
		line += string(buf)
		if !isP {
			break
		}
	}
	return
}

func Probe(path string) (info *Info, err error) {
	cmd := exec.Command("ffprobe", "-show_format", "-show_streams", path)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	r := bufio.NewReader(out)
	info = &Info{}
	defer func() {
		out.Close()
	}()
	defer func() {
		waitErr := cmd.Wait()
		if waitErr != nil {
			err = waitErr
		}
	}()
	for {
		line, err := readLine(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch line {
		case "[FORMAT]":
			info.Format, err = readSection(r, "[/FORMAT]")
			if err != nil {
				return nil, err
			}
		case "[STREAM]":
			m, err := readSection(r, "[/STREAM]")
			if err != nil {
				return nil, err
			}
			var i int
			if _, err := fmt.Sscan(m["index"], &i); err != nil {
				return nil, err
			}
			if i != len(info.Streams) {
				return nil, errors.New("streams unordered")
			}
			info.Streams = append(info.Streams, m)
		default:
			return nil, errors.New(fmt.Sprint("unknown section:", line))
		}
	}
	return info, nil
}
