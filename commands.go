package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/urfave/cli"
)

type DongleConfig map[string]struct {
	IMEI string `json:"imei"`
	IMSI string `json:"imsi"`
}

func ToAST(c DongleConfig) *Ast {
	a := &Ast{}
	for k, v := range c {
		s := &NodeSection{name: k}
		s.values = append(s.values, &nodeIdent{
			key:   "imei",
			value: v.IMEI,
		})
		s.values = append(s.values, &nodeIdent{
			key:   "imsi",
			value: v.IMSI,
		})
		a.Sections = append(a.Sections, s)
	}
	return a
}

func Dongles(ctx *cli.Context) error {
	src := ctx.Args().First()
	var b []byte
	var err error
	if src == "stdin" {
		b, err = ReadFromStdin()
		if err != nil {
			return err
		}
	} else {
		if src == "" {
			return errors.New("either supply a config file or pip stuff to stdin")
		}
		b, err = ioutil.ReadFile(src)
		if err != nil {
			return err
		}
	}
	c := make(DongleConfig)
	err = json.Unmarshal(b, &c)
	if err != nil {
		return err
	}
	a := ToAST(c)
	o, err := PatchAst(a)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	PrintAst(&buf, o)
	fmt.Println(&buf)
	return nil
}

func PatchAst(dst *Ast) (*Ast, error) {
	name := filepath.Join(asteriskDir(), "modem.conf")
	b, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}
	p, err := NewParser(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	a, err := p.Parse()
	if err != nil {
		return nil, err
	}

	patch := &Ast{}
	for _, s := range a.Sections {
		for _, v := range dst.Sections {
			if v.name == s.name {
				patch.Sections = append(patch.Sections, v)
			} else {
				patch.Sections = append(patch.Sections, s)
			}
		}
	}

	var buf bytes.Buffer
	PrintAst(&buf, patch)
	fmt.Println(&buf)
	o := &Ast{}
	fmt.Println(len(patch.Sections))
	for _, v := range patch.Sections {
		for _, i := range v.values {
			if i.key == "imei" {
				if n := byIMEI(dst, i.value); n != nil {
					if n.name == v.name {
						continue
					}
					o.Sections = append(o.Sections, n)
					continue
				}
			}
		}
		o.Sections = append(o.Sections, v)
	}
	fmt.Println("HERE")
	return o, nil
}

func byIMEI(a *Ast, imei string) *NodeSection {
	for _, s := range a.Sections {
		for _, v := range s.values {
			if v.key == "imei" && v.value == imei {
				return s
			}
		}
	}
	return nil
}
func bySection(a *Ast, name string) *NodeSection {
	for _, s := range a.Sections {
		if s.name == name {
			return s
		}
	}
	return nil
}

func asteriskDir() string {
	return os.Getenv("ASTERISK_CONFIG")
}

func ReadFromStdin() ([]byte, error) {
	r := bufio.NewReader(os.Stdin)
	return r.ReadBytes('\n')
}
