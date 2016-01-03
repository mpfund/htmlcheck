package htmlcheck

import (
	"os"
	"testing"
)

var v Validator = Validator{}

func TestMain(m *testing.M) {
	v.AddValidTag(ValidTag{
		Name:          "a",
		Attrs:         []string{"href"},
		IsSelfClosing: true,
	})
	v.AddValidTag(ValidTag{
		Name:          "b",
		Attrs:         []string{"id"},
		IsSelfClosing: false,
	})
	v.AddValidTag(ValidTag{
		Name:          "c",
		Attrs:         []string{"id"},
		IsSelfClosing: false,
	})
	v.AddValidTag(ValidTag{
		Name:          "style",
		Attrs:         []string{"id"},
		IsSelfClosing: false,
	})
	os.Exit(m.Run())
}

func checkErrors(t *testing.T, errors []*ValidationError) {
	if len(errors) > 0 {
		t.Fatal(errors)
	}
}

func hasErrors(t *testing.T, errors []*ValidationError, text string) {
	if len(errors) == 0 {
		t.Fatal("should raise error: " + text)
	}
	t.Log(errors)
}

func Test_SingleTag(t *testing.T) {
	errors := v.ValidateHtmlString("<a></a>")
	checkErrors(t, errors)
}

func Test_SelfClosingTag(t *testing.T) {
	errors := v.ValidateHtmlString("<a>")
	checkErrors(t, errors)
}

func Test_SingleAttr(t *testing.T) {
	errors := v.ValidateHtmlString("<a href='test'>")
	checkErrors(t, errors)
}

func Test_UnknownAttr(t *testing.T) {
	errors := v.ValidateHtmlString("<a hrefff='test'>")
	hasErrors(t, errors, "invalid attribute")
}

func Test_DuplicatedAttr(t *testing.T) {
	errors := v.ValidateHtmlString("<a href='test' href='test2'>")
	hasErrors(t, errors, "duplicated attribute")
}

func Test_SingleUnknownTag(t *testing.T) {
	errors := v.ValidateHtmlString("<art>")
	hasErrors(t, errors, "tag unkown")
}

func Test_UnclosedTag(t *testing.T) {
	errors := v.ValidateHtmlString("<b>df")
	hasErrors(t, errors, "tag unclosed")
}

func Test_NestedTags(t *testing.T) {
	errors := v.ValidateHtmlString("<b><a></a></b>")
	checkErrors(t, errors)
}

func Test_WronglyNestedTags(t *testing.T) {
	errors := v.ValidateHtmlString("<b><c></b></c>")
	hasErrors(t, errors, "b closed before opended")
}

func Test_SwapedStartClosingTags(t *testing.T) {
	errors := v.ValidateHtmlString("</b><b>")
	hasErrors(t, errors, "b closed before opended")
}

func Test_NextedTagsWithSelfClosing(t *testing.T) {
	errors := v.ValidateHtmlString("<b><a></b>")
	checkErrors(t, errors)
}

func Test_NextedTagsWithUnkonwAttribute1(t *testing.T) {
	errors := v.ValidateHtmlString("<b kkk='kkk'><a></b>")
	if len(errors) != 1 {
		t.Fatal("should raise invalid attribute error")
	}
}

func Test_NextedTagsWithUnkonwAttribute2(t *testing.T) {
	errors := v.ValidateHtmlString("<b><a kkk='kkk'></b>")
	if len(errors) != 1 {
		t.Fatal("should raise invalid attribute error")
	}
}

func Test_LineColumn_SingleLine(t *testing.T) {
	errors := v.ValidateHtmlString("<b><a kkk='kkk'></b>")
	if errors[0].TextPos.Line != 1 {
		t.Fatal(errors[0].TextPos)
	}
	if errors[0].TextPos.Column != 5 {
		t.Fatal(errors[0].TextPos)
	}
}

func Test_LineColumn_MultipleLines(t *testing.T) {
	errors := v.ValidateHtmlString("<b></b>\n<b></b>\n<b kkk='kkk'></b>")
	if errors[0].TextPos.Line != 3 {
		t.Fatal(errors[0].TextPos)
	}
	if errors[0].TextPos.Column != 2 {
		t.Fatal(errors[0].TextPos)
	}
}

func Test_IsValidAttribute(t *testing.T) {
	ok := v.IsValidAttribute("a", "href")
	if !ok {
		t.Fatal("should return true")
	}
	ok = v.IsValidAttribute("kkk", "href")
	if ok {
		t.Fatal("should return false")
	}
}

func Test_Callback(t *testing.T) {
	triggerd := false
	v.RegisterCallback(func(tagName string, attributeName string,
		value string, reason ErrorReason) *ValidationError {
		triggerd = true
		return nil
	})

	errors := v.ValidateHtmlString("<kk>")
	if !triggerd {
		t.Fatal("should trigger callback")
	}

	checkErrors(t, errors)
}
