package htmlcheck

import (
	"bytes"
	"errors"
	"golang.org/x/net/html"
	"io"
)

type Invalidation int

const (
	InvTag                 Invalidation = 0
	InvAttribute           Invalidation = 1
	InvClosedBeforeOpened  Invalidation = 2
	InvNotProperlyClosed   Invalidation = 3
	InvDuplicatedAttribute Invalidation = 4
)

type InvalidationCallback func(tagName string, attributeName string, value string, reason Invalidation) error

type ValidTag struct {
	Name          string
	Attrs         []string
	IsSelfClosing bool
}

type Validator struct {
	validTagMap          map[string]map[string]bool
	validSelfClosingTags map[string]bool
	invalidationCallback InvalidationCallback
}

func (v *Validator) AddValidTags(validTags []ValidTag) {
	v.validSelfClosingTags = make(map[string]bool)
	v.validTagMap = make(map[string]map[string]bool)

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

func (v *Validator) RegisterCallback(f InvalidationCallback) {
	v.invalidationCallback = f
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

func (v *Validator) ValidateHtmlString(str string) error {
	buffer := bytes.NewBuffer([]byte(str))
	return v.ValidateHtml(buffer)
}

func (v *Validator) checkInvalidationCallback(tagName string, attr string, value string, reason Invalidation) error {
	if v.invalidationCallback != nil {
		return v.invalidationCallback(tagName, attr, value, reason)
	}
	return GetError(tagName, attr, reason)
}

func GetError(tagName string, attrName string, i Invalidation) error {
	switch i {
	case InvTag:
		return errors.New("tag '" + tagName + "' is not valid")
	case InvAttribute:
		return errors.New("invalid attribute '" + attrName + "' in tag '" + tagName + "'")
	case InvClosedBeforeOpened:
		return errors.New("close tag '" + tagName + "' was not opened before close tag.")
	case InvNotProperlyClosed:
		return errors.New("tag '" + tagName + "' is not properly closed")
	case InvDuplicatedAttribute:
		return errors.New("duplicated attribute '" + attrName + "' in '" + tagName + "'")
	}
	return nil
}

func (v *Validator) ValidateHtml(r io.Reader) error {
	d := html.NewTokenizer(r)
	var openClosedCount = make(map[string]int)

	for {
		// token type
		tokenType := d.Next()
		if tokenType == html.ErrorToken {
			if d.Err() == io.EOF {
				break
			}

			return d.Err()
		}
		token := d.Token()
		if tokenType == html.EndTagToken ||
			tokenType == html.StartTagToken ||
			tokenType == html.SelfClosingTagToken {

			tagName := token.Data

			if !v.IsValidTag(tagName) {
				callbackerr := v.checkInvalidationCallback(tagName, "", "", InvTag)
				if callbackerr != nil {
					return callbackerr
				}
			}

			attrs := make(map[string]bool)

			for _, attr := range token.Attr {
				if !v.IsValidAttribute(tagName, attr.Key) {
					callbackerr := v.checkInvalidationCallback(tagName, attr.Key, attr.Val, InvTag)
					if callbackerr != nil {
						return callbackerr
					}
				}
				_, ok := attrs[attr.Key]
				if !ok {
					attrs[attr.Key] = true
				} else {
					callbackerr := v.checkInvalidationCallback(tagName, attr.Key, attr.Val, InvDuplicatedAttribute)
					if callbackerr != nil {
						return callbackerr
					}
				}
			}

			i, ok := openClosedCount[tagName]
			if token.Type == html.StartTagToken || token.Type == html.SelfClosingTagToken {
				if ok {
					openClosedCount[tagName] = i + 1
				} else {
					openClosedCount[tagName] = 1
				}
			}
			if token.Type == html.EndTagToken {
				if ok {
					openClosedCount[tagName] = i - 1
				} else {
					if tokenType == html.EndTagToken {
						callbackerr := v.checkInvalidationCallback(tagName, "", "", InvClosedBeforeOpened)
						if callbackerr != nil {
							return callbackerr
						}
					}
				}
			}
		}
	}

	for tagName, m := range openClosedCount {
		if m > 0 && !v.IsValidSelfClosingTag(tagName) {
			callbackerr := v.checkInvalidationCallback(tagName, "", "", InvNotProperlyClosed)
			if callbackerr != nil {
				return callbackerr
			}
		}
	}
	return nil
}
