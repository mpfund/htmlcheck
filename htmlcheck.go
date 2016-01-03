package htmlcheck

import (
	"fmt"
	"golang.org/x/net/html"
	"io"
	"strings"
)

type ErrorReason int

const (
	InvTag                 ErrorReason = 0
	InvAttribute           ErrorReason = 1
	InvClosedBeforeOpened  ErrorReason = 2
	InvNotProperlyClosed   ErrorReason = 3
	InvDuplicatedAttribute ErrorReason = 4
)

type ErrorCallback func(tagName string, attributeName string,
	value string, reason ErrorReason) *ValidationError

type ValidTag struct {
	Name          string
	Attrs         []string
	IsSelfClosing bool
}

type ValidationError struct {
	TagName       string
	AttributeName string
	Reason        ErrorReason
}

func (e *ValidationError) Error() string {
	switch e.Reason {
	case InvTag:
		return "tag '" + e.TagName + "' is not valid"
	case InvAttribute:
		return "invalid attribute '" + e.AttributeName + "' in tag '" + e.TagName + "'"
	case InvClosedBeforeOpened:
		return "'" + e.TagName + "' closed before opened."
	case InvNotProperlyClosed:
		return "tag '" + e.TagName + "' is never closed"
	case InvDuplicatedAttribute:
		return "duplicated attribute '" + e.AttributeName + "' in '" + e.TagName + "'"
	}
	return "unknown error"
}

type Validator struct {
	validTagMap          map[string]map[string]bool
	validSelfClosingTags map[string]bool
	errorCallback        ErrorCallback
	StopAfterFirstError  bool
}

func (v *Validator) AddValidTags(validTags []ValidTag) {
	if v.validSelfClosingTags == nil {
		v.validSelfClosingTags = make(map[string]bool)
	}
	if v.validTagMap == nil {
		v.validTagMap = make(map[string]map[string]bool)
	}

	for _, tag := range validTags {
		if tag.IsSelfClosing {
			v.validSelfClosingTags[tag.Name] = true
		}
		v.validTagMap[tag.Name] = make(map[string]bool)
		for _, a := range tag.Attrs {
			v.validTagMap[tag.Name][a] = true
		}
	}
}

func (v *Validator) AddValidTag(validTag ValidTag) {
	v.AddValidTags([]ValidTag{validTag})
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
	attrs, ok := v.validTagMap[tagName]
	if !ok {
		return false
	}
	_, ok = attrs[attrName]
	return ok
}

func (v *Validator) ValidateHtmlString(str string) []*ValidationError {
	buffer := strings.NewReader(str)
	return v.ValidateHtml(buffer)
}

func (v *Validator) checkErrorCallback(tagName string, attr string,
	value string, reason ErrorReason) *ValidationError {
	if v.errorCallback != nil {
		return v.errorCallback(tagName, attr, value, reason)
	}
	return &ValidationError{tagName, attr, reason}
}

func (v *Validator) ValidateHtml(r io.Reader) []*ValidationError {
	d := html.NewTokenizer(r)

	parents := []string{}
	var err *ValidationError
	errors := []*ValidationError{}
	for {
		// token type
		tokenType := d.Next()
		if tokenType == html.ErrorToken {
			break
		}
		token := d.Token()
		parents, err = v.checkToken(tokenType, token, parents)
		if err != nil {
			errors = append(errors, err)
			if v.StopAfterFirstError {
				return errors
			} else {
				//parents = v.correctError(err, parents, tokenType, token)
			}
		}
	}

	err = v.checkParents(parents)
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

func (v *Validator) checkParents(parents []string) *ValidationError {
	for _, tagName := range parents {
		if v.IsValidSelfClosingTag(tagName) {
			continue
		}
		cError := v.checkErrorCallback(tagName, "", "", InvNotProperlyClosed)
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

func (v *Validator) checkToken(tokenType html.TokenType, token html.Token,
	parents []string) ([]string, *ValidationError) {

	if tokenType == html.EndTagToken ||
		tokenType == html.StartTagToken ||
		tokenType == html.SelfClosingTagToken {

		tagName := token.Data

		if !v.IsValidTag(tagName) {
			cError := v.checkErrorCallback(tagName, "", "", InvTag)
			if cError != nil {
				return parents, cError
			}
		}

		attrs := map[string]bool{}

		for _, attr := range token.Attr {
			if !v.IsValidAttribute(tagName, attr.Key) {
				cError := v.checkErrorCallback(tagName, attr.Key,
					attr.Val, InvAttribute)
				if cError != nil {
					return parents, cError
				}
			}
			_, ok := attrs[attr.Key]
			if !ok {
				attrs[attr.Key] = true
			} else {
				cError := v.checkErrorCallback(tagName, attr.Key,
					attr.Val, InvDuplicatedAttribute)
				if cError != nil {
					return parents, cError
				}
			}
		}

		if token.Type == html.StartTagToken ||
			token.Type == html.SelfClosingTagToken {
			parents = append(parents, tagName)
		}

		if token.Type == html.EndTagToken {
			if len(parents) > 0 && parents[len(parents)-1] == tagName {
				parents = popLast(parents)
			} else if parents[len(parents)-1] != tagName ||
				len(parents) == 0 {
				index := indexOf(parents, tagName)
				if index > -1 {
					missingTagName := parents[len(parents)-1]
					cError := v.checkErrorCallback(missingTagName,
						"", "", InvNotProperlyClosed)
					parents = parents[0:index]
					if cError != nil {
						return parents, cError
					}
				} else {
					cError := v.checkErrorCallback(tagName,
						"", "", InvClosedBeforeOpened)
					if cError != nil {
						return parents, cError
					}
				}
			}
		}
	}

	return parents, nil
}
