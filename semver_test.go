package main

import "testing"

func Test_ValidSemVer(t *testing.T) {
	parsed, err := parseSemanticVersion("1.2.3")
	if err != nil {
		t.Fail()
	}
	if parsed.major != 1 || parsed.minor != 2 || parsed.patch != 3 {
		t.Fail()
	}
}

func Test_InvalidSemVer(t *testing.T) {
	_, err := parseSemanticVersion("22.2")
	if err == nil {
		t.Fail()
	}
}

func Test_SemVerParserIgnoresJunk(t *testing.T) {
	ok, err := isVersionCompatible("#kallekula: ^1.2.3", "v1.2.3")
	if err != nil {
		t.Fail()
	}
	if !ok {
		t.Fail()
	}
}

func Test_EqualSemVer(t *testing.T) {
	ok, err := isVersionCompatible("=1.2.3", "v1.2.3")
	if err != nil {
		t.Fail()
	}
	if !ok {
		t.Fail()
	}
}

func Test_TildeSemVerEqual(t *testing.T) {
	ok, err := isVersionCompatible("~1.2.3", "v1.2.3")
	if err != nil {
		t.Fail()
	}
	if !ok {
		t.Fail()
	}
}

func Test_TildeSemVerPatchOK(t *testing.T) {
	ok, err := isVersionCompatible("~1.2.3", "v1.2.4")
	if err != nil {
		t.Fail()
	}
	if !ok {
		t.Fail()
	}
}

func Test_TildeSemVerPatchNOK(t *testing.T) {
	ok, err := isVersionCompatible("~1.2.3", "v1.2.2")
	if err != nil {
		t.Fail()
	}
	if ok {
		t.Fail()
	}
}

func Test_TildeSemVerMinorNOK(t *testing.T) {
	ok, err := isVersionCompatible("~1.2.3", "v1.3.4")
	if err != nil {
		t.Fail()
	}
	if ok {
		t.Fail()
	}
}

func Test_TildeSemVerMajorNOK(t *testing.T) {
	ok, err := isVersionCompatible("~1.2.3", "v0.3.4")
	if err != nil {
		t.Fail()
	}
	if ok {
		t.Fail()
	}
}

func Test_CaretSemVerEqual(t *testing.T) {
	ok, err := isVersionCompatible("^1.2.3", "v1.2.3")
	if err != nil {
		t.Fail()
	}
	if !ok {
		t.Fail()
	}
}

func Test_CaretSemVerMinorOK(t *testing.T) {
	ok, err := isVersionCompatible("^1.2.3", "v1.4.5")
	if err != nil {
		t.Fail()
	}
	if !ok {
		t.Fail()
	}
}

func Test_CaretSemVerMinorNOK(t *testing.T) {
	ok, err := isVersionCompatible("^1.2.3", "v1.1.5")
	if err != nil {
		t.Fail()
	}
	if ok {
		t.Fail()
	}
}

func Test_CaretSemVerPatchOK(t *testing.T) {
	ok, err := isVersionCompatible("^1.2.3", "v1.2.5")
	if err != nil {
		t.Fail()
	}
	if !ok {
		t.Fail()
	}
}

func Test_CaretSemVerPatchNOK(t *testing.T) {
	ok, err := isVersionCompatible("^1.2.3", "v1.2.1")
	if err != nil {
		t.Fail()
	}
	if ok {
		t.Fail()
	}
}

func Test_CaretSemVerMajorNOK(t *testing.T) {
	ok, err := isVersionCompatible("~1.2.3", "v0.3.4")
	if err != nil {
		t.Fail()
	}
	if ok {
		t.Fail()
	}
}
