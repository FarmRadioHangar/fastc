package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Ast is an abstract syntax tree for a scanneruration object. The scanneruration
// format should be section based( or you can say namespacing).
type Ast struct {
	Sections []*NodeSection
}

//Section returns the section named name or an error if the section is not found
//in the Ast
func (a *Ast) Section(name string) (*NodeSection, error) {
	for _, v := range a.Sections {
		if v.name == name {
			return v, nil
		}
	}
	return nil, errors.New("section not found")
}

//ToJSON marhalls *Ast to a json string and writes the result to dst
func (a *Ast) ToJSON(dst io.Writer) error {
	o := make(map[string]interface{})
	for _, v := range a.Sections {
		sec := make(map[string]interface{})
		for _, value := range v.values {
			sec[value.key] = value.value
		}
		o[v.name] = sec
	}
	return json.NewEncoder(dst).Encode(o)
}

//LoadJSON loads AST from json src
func (a *Ast) LoadJSON(src []byte) error {
	var obj map[string]interface{}
	err := json.Unmarshal(src, &obj)
	if err != nil {
		return err
	}
	for key, value := range obj {
		ns := &NodeSection{name: key}
		switch value.(type) {
		case map[string]interface{}:
			sec := value.(map[string]interface{})
			for k, v := range sec {
				ident := &nodeIdent{}
				ident.key = k
				ident.value = fmt.Sprint(v)
				ns.values = append(ns.values, ident)
			}
		}
		a.Sections = append(a.Sections, ns)
	}
	return nil
}

func PrintAst(dst io.Writer, src *Ast) {
	for _, v := range src.Sections {
		if v.name == "main" {
			fmt.Fprintf(dst, "\n\n")
			for _, sub := range v.values {
				fmt.Fprintf(dst, "%s=%s \n", sub.key, sub.value)
			}
			fmt.Fprint(dst, "\n\n")
			continue
		}
		fmt.Fprintf(dst, "\n [%s]\n", v.name)
		for _, sub := range v.values {
			fmt.Fprintf(dst, "%s=%s \n", sub.key, sub.value)
		}
		fmt.Fprint(dst, "\n\n")
		continue
	}
}

//NodeSection represent a section in the scanneruration object. Sections are name
//spaces that contains scannerurations definitions under them.
type NodeSection struct {
	name   string
	line   int
	values []*nodeIdent
}

//Get access the key definition and returns its value or an error if the key is
//not part of the section.
func (n *NodeSection) Get(key string) (string, error) {
	for _, v := range n.values {
		if v.key == key {
			return v.value, nil
		}
	}
	return "", errors.New("key not found")
}

//nodeIdent represents a scanneruration definition, which can be the key value
//definition.
type nodeIdent struct {
	key   string
	value string
	line  int
}

// Parser is a Parser for scanneruration files. It supports utf-8 encoded
// scanneruration files.
//
// Only modem scanneruration files are supported for the momment.
type Parser struct {
	tokens  []*Token
	Ast     *Ast
	currPos int
}

//NewParser returns a new Parser that parses input from src. The returned Parser
//supports gsm modem scanneruration format only.
func NewParser(src io.Reader) (*Parser, error) {
	s := NewScanner(src)
	var toks []*Token
	var err error
	var tok *Token
	for err == nil {
		tok, err = s.Scan()
		if err != nil {
			if err.Error() != io.EOF.Error() {
				return nil, err
			}

			// we are at the end of the input
			break
		}
		if tok != nil {
			switch tok.Type {
			case WhiteSpace, Comment:

				// Skip comments and whitespaces but preserve the newlines to aid in
				// parsing
				continue
			default:
				toks = append(toks, tok)
			}

		}
	}
	return &Parser{tokens: toks, Ast: &Ast{}}, nil
}

// Parse parses the scanned input and return its *Ast or arror if any.
func (p *Parser) Parse() (*Ast, error) {
	var err error
	if err != nil {
		return nil, err
	}
	mainSec := &NodeSection{name: "main"}
END:
	for {
		tok := p.next()
		if tok.Type == EOF {
			break END
		}
		switch tok.Type {
		case LBrace:
			p.rewind()
			err = p.parseSection()
			if err != nil {
				break END
			}
		case Ident:
			p.rewind()
			err = p.parseIdent(mainSec)
			if err != nil {
				break END
			}

		}
	}
	if err != nil {
		return nil, err
	}
	p.Ast.Sections = append([]*NodeSection{mainSec}, p.Ast.Sections...)
	return p.Ast, err
}

func (p *Parser) next() *Token {
	if p.currPos >= len(p.tokens)-1 {
		return &Token{Type: EOF}
	}
	t := p.tokens[p.currPos]
	p.currPos++
	return t
}

func (p *Parser) seek(at int) {
	p.currPos = at
}

func (p *Parser) parseSection() (err error) {
	left := p.next()
	if left.Type != LBrace {
		return errors.New("bad token")
	}
	ns := &NodeSection{}
	completeName := false
END:
	for {
	BEGIN:
		tok := p.next()
		if tok.Type == EOF {
			p.rewind()
			break END
		}

		if !completeName {
			switch tok.Type {
			case Ident:
				ns.name = ns.name + tok.Text
				goto BEGIN
			case RBrace:
				completeName = true
				goto BEGIN
			}
		}
		switch tok.Type {
		case NLine:
			n1 := p.next()
			if n1.Type == NLine {
				n2 := p.next()
				if n2.Type == NLine {
					break END
				}
				p.rewind()
				goto BEGIN
			}
			goto BEGIN
		case Ident:
			p.rewind()
			err = p.parseIdent(ns)
			if err != nil {
				break END
			}
		default:
			break END
		}
	}
	if err == nil {
		p.Ast.Sections = append(p.Ast.Sections, ns)
	}
	return
}

func (p *Parser) rewind() {
	p.currPos--
}

func (p *Parser) parseIdent(sec *NodeSection) (err error) {
	n := &nodeIdent{}
	doneKey := false
END:
	for {
	BEGIN:
		tok := p.next()
		if tok.Type == EOF {
			p.rewind()
			break END
		}

		if !doneKey {
			switch tok.Type {
			case Ident:
				n.key = n.key + tok.Text
				goto BEGIN
			case Assign:
				doneKey = true
				goto BEGIN
			default:
				err = errors.New("some fish")
				break END
			}

		}
		switch tok.Type {
		case Ident:
			n.value = n.value + tok.Text
			goto BEGIN
		case NLine:
			break END
		default:
			err = errors.New("some fish")
			break END
		}
	}
	if err == nil {
		sec.values = append(sec.values, n)
	}

	return
}
