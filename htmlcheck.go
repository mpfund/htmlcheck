package htmlcheck

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"

	//"golang.org/x/net/html"
	html "github.com/BlackEspresso/htmlcheck/htmlp"
)

type ErrorReason int

const (
	InvTag                 ErrorReason = 0
	InvAttribute           ErrorReason = 1
	InvClosedBeforeOpened  ErrorReason = 2
	InvNotProperlyClosed   ErrorReason = 3
	InvDuplicatedAttribute ErrorReason = 4
	InvEOF                 ErrorReason = 5
)

type Span struct {
	Start int
	End   int
}

type TextPos struct {
	Line   int
	Column int
}

type ErrorCallback func(tagName string, attributeName string,
	value string, reason ErrorReason) *ValidationError

type ValidTag struct {
	Name          string
	Attrs         []string
	AttrRegEx     string
	IsSelfClosing bool
}

type ValidationError struct {
	TagName       string
	AttributeName string
	Reason        ErrorReason
	Pos           Span
	TextPos       *TextPos
}

func (e *ValidationError) Error() string {
	text := ""
	switch e.Reason {
	case InvTag:
		text = "tag '" + e.TagName + "' is not valid"
	case InvAttribute:
		text = "invalid attribute '" + e.AttributeName + "' in tag '" + e.TagName + "'"
	case InvClosedBeforeOpened:
		text = "'" + e.TagName + "' closed before opened."
	case InvNotProperlyClosed:
		text = "tag '" + e.TagName + "' is never closed"
	case InvDuplicatedAttribute:
		text = "duplicated attribute '" + e.AttributeName + "' in '" + e.TagName + "'"
	}

	pos := ""

	start := strconv.Itoa(e.Pos.Start)
	end := strconv.Itoa(e.Pos.End)
	pos = " (" + start + ", " + end + ")"

	if e.TextPos == nil {

	} else {
		line := strconv.Itoa(e.TextPos.Line)
		column := strconv.Itoa(e.TextPos.Column)
		pos = pos + " (L" + line + ", C" + column + ")"
	}

	return text + pos
}

type Validator struct {
	validTagMap          map[string]map[string]bool
	validSelfClosingTags map[string]bool
	errorCallback        ErrorCallback
	StopAfterFirstError  bool
	validTags            map[string]*ValidTag
}

func (v *Validator) AddValidTags(validTags []*ValidTag) {
	if v.validSelfClosingTags == nil {
		v.validSelfClosingTags = make(map[string]bool)
	}
	if v.validTagMap == nil {
		v.validTagMap = make(map[string]map[string]bool)
	}
	if v.validTags == nil {
		v.validTags = map[string]*ValidTag{}
	}

	for _, tag := range validTags {
		if tag.IsSelfClosing {
			v.validSelfClosingTags[tag.Name] = true
		}
		v.validTagMap[tag.Name] = make(map[string]bool)
		for _, a := range tag.Attrs {
			v.validTagMap[tag.Name][a] = true
		}
		if tag.Name == "" {
			_, hasGlobalTag := v.validTags[""]
			if hasGlobalTag {
				log.Println("second global tag")
			}
		}
		v.validTags[tag.Name] = tag
	}
}

func (v *Validator) AddValidTag(validTag ValidTag) {
	v.AddValidTags([]*ValidTag{&validTag})
}

func (v *Validator) RegisterCallback(f ErrorCallback) {
	v.errorCallback = f
}

func (v *Validator) IsValidTag(tagName string) bool {
	_, ok := v.validTagMap[tagName]
	return ok
}

func (v *Validator) IsValidSelfClosingTag(tagName string) bool {
	_, ok := v.validSelfClosingTags[tagName]
	if !ok {
		return false
	}
	return ok
}

func (v *Validator) IsValidAttribute(tagName string, attrName string) bool {
	attrs, hasTag := v.validTagMap[tagName]
	gAttrs, hasGlobals := v.validTagMap[""] //check global attributes

	if hasGlobals {
		_, hasGlobalAttr := gAttrs[attrName]
		if hasGlobalAttr {
			return true
		} else {
			//test reg ex
			tag := v.validTags[""]
			if tag.AttrRegEx != "" {
				matches, err := regexp.MatchString(tag.AttrRegEx, attrName)
				if err == nil && matches {
					return true
				}
			}
		}
	}

	if hasTag {
		_, hasAttr := attrs[attrName]
		if hasAttr {
			return true
		} else {
			//test reg ex
			tag := v.validTags[tagName]
			if tag.AttrRegEx != "" {
				matches, err := regexp.MatchString(tag.AttrRegEx, attrName)
				if err == nil && matches {
					return true
				}
			}
		}
	}

	return false
}

func (v *Validator) ValidateHtmlString(str string) []*ValidationError {
	buffer := strings.NewReader(str)
	errors := v.ValidateHtml(buffer)
	updateLineColumns(str, errors)
	return errors
}

func updateLineColumns(str string, errors []*ValidationError) {
	lines := strings.Split(str, "\n")
	for _, k := range errors {
		charCount := 0
		for i, l := range lines {
			lineLen := len(l) + 1
			if k.Pos.Start < (charCount + lineLen) {
				tPos := TextPos{i + 1, k.Pos.Start - charCount + 1}
				k.TextPos = &tPos
				break
			}
			charCount += lineLen
		}
	}
}

func (v *Validator) checkErrorCallback(tagName string, attr string,
	value string, span Span, reason ErrorReason) *ValidationError {
	if v.errorCallback != nil {
		return v.errorCallback(tagName, attr, value, reason)
	}
	return &ValidationError{tagName, attr, reason, span, nil}
}

func (v *Validator) ValidateHtml(r io.Reader) []*ValidationError {
	d := html.NewTokenizer(r)
	parents := []string{}
	var err *ValidationError
	errors := []*ValidationError{}
	for {
		parents, err = v.checkToken(d, parents)

		if err != nil {
			if err.Reason == InvEOF {
				break
			}
			errors = append(errors, err)
			if v.StopAfterFirstError {
				return errors
			}
		}
	}

	err = v.checkParents(d, parents)
	if err != nil {
		errors = append(errors, err)
	}
	return errors
}

func indexOf(arr []string, val string) int {
	for i, k := range arr {
		if k == val {
			return i
		}
	}
	return -1
}

func (v *Validator) correctError(err *ValidationError, parents []string,
	tokenType html.TokenType, token html.Token) []string {
	if err.Reason == InvClosedBeforeOpened && tokenType == html.EndTagToken {
		index := indexOf(parents, token.Data)
		if index > -1 {
			parents = parents[0:index]
		}
	}
	fmt.Println("correct", parents, tokenType, token.Data)
	return parents
}

func (v *Validator) checkParents(d *html.Tokenizer, parents []string) *ValidationError {
	for _, tagName := range parents {
		if v.IsValidSelfClosingTag(tagName) {
			continue
		}

		pos := getPosition(d)
		cError := v.checkErrorCallback(tagName, "", "", pos, InvNotProperlyClosed)
		if cError != nil {
			return cError
		}
	}
	return nil
}

func popLast(list []string) []string {
	if len(list) == 0 {
		return list
	}
	return list[0 : len(list)-1]
}

func getPosition(d *html.Tokenizer) Span {
	posStart, posEnd := d.GetRawPosition()
	return Span{posStart, posEnd}
}

func (v *Validator) checkToken(d *html.Tokenizer,
	parents []string) ([]string, *ValidationError) {

	tokenType := d.Next()

	if tokenType == html.ErrorToken {
		return parents, &ValidationError{"", "", InvEOF, Span{0, 0}, nil}
	}

	pos := getPosition(d)
	token := d.Token()
	//pos := getPosition(d)

	if tokenType == html.EndTagToken ||
		tokenType == html.StartTagToken ||
		tokenType == html.SelfClosingTagToken {

		tagName := token.Data

		if !v.IsValidTag(tagName) {
			cError := v.checkErrorCallback(tagName, "", "", pos, InvTag)
			if cError != nil {
				return parents, cError
			}
		}

		if token.Type == html.StartTagToken ||
			token.Type == html.SelfClosingTagToken {
			parents = append(parents, tagName)
		}

		attrs := map[string]bool{}

		for _, attr := range token.Attr {
			if !v.IsValidAttribute(tagName, attr.Key) {
				cError := v.checkErrorCallback(tagName, attr.Key,
					attr.Val, pos, InvAttribute)
				if cError != nil {
					return parents, cError
				}
			}
			_, ok := attrs[attr.Key]
			if !ok {
				attrs[attr.Key] = true
			} else {
				cError := v.checkErrorCallback(tagName, attr.Key,
					attr.Val, pos, InvDuplicatedAttribute)
				if cError != nil {
					return parents, cError
				}
			}
		}

		if token.Type == html.EndTagToken {
			if len(parents) > 0 && parents[len(parents)-1] == tagName {
				parents = popLast(parents)
			} else if len(parents) == 0 ||
				parents[len(parents)-1] != tagName {
				index := indexOf(parents, tagName)
				if index > -1 {
					missingTagName := parents[len(parents)-1]
					parents = parents[0:index]
					if !v.IsValidSelfClosingTag(missingTagName) {
						cError := v.checkErrorCallback(missingTagName,
							"", "", pos, InvNotProperlyClosed)
						if cError != nil {
							return parents, cError
						}
					}
				} else {
					cError := v.checkErrorCallback(tagName,
						"", "", pos, InvClosedBeforeOpened)
					if cError != nil {
						return parents, cError
					}
				}
			}
		}
	}

	return parents, nil
}
