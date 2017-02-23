package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/urfave/cli"
)

const (
	dongleFile = "dongle_fessbox.conf"
	diaplanTpl = "extensions_additional.conf.fastc"
	diaplanOut = "extensions_additional.conf"
)

type DongleConfig map[string]map[string]interface{}

func ToAST(c DongleConfig) *Ast {
	a := &Ast{}
	for k, v := range c {
		s := &NodeSection{name: k}
		if imsi, ok := v["imsi"]; ok {
			s.values = append(s.values, &nodeIdent{
				key:   "imsi",
				value: fmt.Sprint(imsi),
			})
		}
		if rx, ok := v["rx-gain"]; ok {
			s.values = append(s.values, &nodeIdent{
				key:   "rx-gain",
				value: fmt.Sprint(rx),
			})
		}
		if tx, ok := v["tx-gain"]; ok {
			s.values = append(s.values, &nodeIdent{
				key:   "tx-gain",
				value: fmt.Sprint(tx),
			})
		}

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
	// o, err := PatchAst(a)
	// if err != nil {
	// 	return err
	// }
	var buf bytes.Buffer
	PrintAst(&buf, a)
	err = ioutil.WriteFile(filepath.Join(asteriskDir(), dongleFile),
		buf.Bytes(), 0644,
	)
	if err != nil {
		return err
	}
	var tctx []map[string]interface{}
	for _, v := range c {
		if calls, ok := v["calls_out"]; ok {
			not := calls.(string) != "disabled"
			if not {
				v["notDisabled"] = not
			}
		}
		tctx = append(tctx, v)
	}
	return writeDialPlan(&TemplateContext{Dongles: tctx})
}

func PatchAst(dst *Ast) (*Ast, error) {
	name := filepath.Join(asteriskDir(), dongleFile)
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
	o := &Ast{}
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
	if dir := os.Getenv("ASTERISK_CONFIG"); dir != "" {
		return dir
	}
	return "/etc/asterisk"
}

func ReadFromStdin() ([]byte, error) {
	r := bufio.NewReader(os.Stdin)
	return r.ReadBytes('\n')
}

type TemplateContext struct {
	Sip     []map[string]interface{}
	Dongles []map[string]interface{}
}

func (c *TemplateContext) AssgignTrunk(from int) string {
	// start with dongles
	var d []map[string]interface{}
	for _, v := range c.Dongles {
		v["trunkID"] = from
		d = append(d, v)
		from++
	}
	c.Dongles = d

	var s []map[string]interface{}
	for _, v := range c.Sip {
		v["trunkID"] = from
		s = append(s, v)
		from++
	}
	c.Dongles = d
	return ""
}

func astToMap(a *Ast) []map[string]interface{} {
	var o []map[string]interface{}
	for _, s := range a.Sections {
		v := make(map[string]interface{})
		v["name"] = s.name
		for _, vv := range s.values {
			v[vv.key] = vv.value
		}
		v["notDisabled"] = true
		o = append(o, v)
	}
	return o
}

func writeDialPlan(ctx *TemplateContext) error {
	b, err := ioutil.ReadFile(filepath.Join(asteriskDir(), diaplanTpl))
	if err != nil {
		return err
	}
	fm := make(template.FuncMap)
	fm["AssignTrunk"] = ctx.AssgignTrunk
	fm["plain"] = func(s string) template.HTML {
		return template.HTML(s)
	}
	tpl, err := template.New("plan").Funcs(fm).Parse(string(b))
	if err != nil {
		return err
	}
	var o bytes.Buffer
	err = tpl.Execute(&o, ctx)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(asteriskDir(), diaplanOut), o.Bytes(), 0600)
}
