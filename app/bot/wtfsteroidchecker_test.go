package bot

import (
	"testing"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// TestWTFSteroidChecker_Contains check that all possible messages can be recognized correctly
//
// nolint
func TestWTFSteroidChecker_Contains(t *testing.T) {
	type fields struct {
		message string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{name: "WTF!",
			fields: fields{
				message: "WTF!",
			},
			want: true},
		{name: "втф!",
			fields: fields{
				message: "втф!",
			},
			want: true},
		{name: "WT\ufff0F!",
			fields: fields{
				message: "WT￰F!",
			},
			want: true},
		{name: "WtF!",
			fields: fields{
				message: "WtF!",
			},
			want: true},
		{name: "𝀥tf!",
			fields: fields{
				message: "𝀥tf!",
			},
			want: true},
		{name: "ẂTF!",
			fields: fields{
				message: "ẂTF!",
			},
			want: true},
		{name: "W TF!",
			fields: fields{
				message: "W TF!",
			},
			want: true},
		{name: "wtf!",
			fields: fields{
				message: "wtf!",
			},
			want: true},
		{name: "wtf?",
			fields: fields{
				message: "wtf?",
			},
			want: true},
		{name: "🅦🅣ⓕ!",
			fields: fields{
				message: "🅦🅣ⓕ!",
			},
			want: true},
		{name: "w-t-f-!",
			fields: fields{
				message: "w-t-f-!",
			},
			want: true},
		{name: "w;t;f;!",
			fields: fields{
				message: "w;t;f;!",
			},
			want: true},
		{name: "W T F !",
			fields: fields{
				message: "W T F !",
			},
			want: true},
		{name: "W῝🇹🶪Ꝼ!",
			fields: fields{
				message: "W῝🇹🶪Ꝼ!",
			},
			want: true},
		{name: "WTḞ!",
			fields: fields{
				message: "WTḞ!",
			},
			want: true},
		{name: "WTF!",
			fields: fields{
				message: "WTF!",
			},
			want: true},
		{name: "Вот фон! - false",
			fields: fields{
				message: "Вот фон!",
			},
			want: false},
		{name: "W؈T؈F؈!",
			fields: fields{
				message: "W؈T؈F؈!",
			},
			want: true},
		{name: "Что за втф! - false",
			fields: fields{
				message: "Что за втф!",
			},
			want: false},
		{name: "Что за wtf! - false",
			fields: fields{
				message: "Что за wtf!",
			},
			want: false},
		{name: "VVtf!",
			fields: fields{
				message: "VVtf!",
			},
			want: true},
		{name: "¡ɟʇʍ",
			fields: fields{
				message: "¡ɟʇʍ",
			},
			want: true},
		{name: "¿ɟʇʍ",
			fields: fields{
				message: "¿ɟʇʍ",
			},
			want: true},
		{name: "wtᷫ!",
			fields: fields{
				message: "wtᷫ!",
			},
			want: true},
		{name: "wtᷥ!",
			fields: fields{
				message: "wtᷥ!",
			},
			want: true},
		{name: "¡ȸɯʚ",
			fields: fields{
				message: "¡ȸɯʚ",
			},
			want: true},
		{name: "¿ȸɯʚ",
			fields: fields{
				message: "¿ȸɯʚ",
			},
			want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WTFSteroidChecker{
				message: tt.fields.message,
			}
			if got := w.Contains(); got != tt.want {
				t.Errorf("WTFSteroidChecker.Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWTFSteroidChecker_ContainsWTF(t *testing.T) {
	type fields struct {
		message string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{name: "WTF",
			fields: fields{
				message: "WTF",
			},
			want: true},
		{name: "втф",
			fields: fields{
				message: "втф",
			},
			want: true},
		{name: "WT\ufff0F",
			fields: fields{
				message: "WT￰F",
			},
			want: true},
		{name: "WtF",
			fields: fields{
				message: "WtF",
			},
			want: true},
		{name: "𝀥tf",
			fields: fields{
				message: "𝀥tf",
			},
			want: true},
		{name: "ẂTF",
			fields: fields{
				message: "ẂTF",
			},
			want: true},
		{name: "W TF",
			fields: fields{
				message: "W TF",
			},
			want: true},
		{name: "wtf",
			fields: fields{
				message: "wtf",
			},
			want: true},
		{name: "wtf",
			fields: fields{
				message: "wtf",
			},
			want: true},
		{name: "🅦🅣ⓕ",
			fields: fields{
				message: "🅦🅣ⓕ",
			},
			want: true},
		{name: "w-t-f",
			fields: fields{
				message: "w-t-f",
			},
			want: true},
		{name: "w;t;f",
			fields: fields{
				message: "w;t;f",
			},
			want: true},
		{name: "W T F",
			fields: fields{
				message: "W T F",
			},
			want: true},
		{name: "W῝🇹🶪Ꝼ",
			fields: fields{
				message: "W῝🇹🶪Ꝼ",
			},
			want: true},
		{name: "WTḞ",
			fields: fields{
				message: "WTḞ",
			},
			want: true},
		{name: "W\x05TF",
			fields: fields{
				message: "W\x05TF",
			},
			want: true},
		{name: "Вот фон! - false",
			fields: fields{
				message: "Вот фон!",
			},
			want: false},
		{name: "W؈T؈F؈",
			fields: fields{
				message: "W؈T؈F؈",
			},
			want: true},
		{name: "Что за втф - true",
			fields: fields{
				message: "Что за втф",
			},
			want: true},
		{name: "Что за wtf - true",
			fields: fields{
				message: "Что за wtf",
			},
			want: true},
		{name: "VVtf",
			fields: fields{
				message: "VVtf",
			},
			want: true},
		{name: "¡ɟʇʍ",
			fields: fields{
				message: "¡ɟʇʍ",
			},
			want: false},
		{name: "¿ɟʇʍ",
			fields: fields{
				message: "¿ɟʇʍ",
			},
			want: false},
		{name: "wtᷫ",
			fields: fields{
				message: "wtᷫ",
			},
			want: true},
		{name: "wtᷥ",
			fields: fields{
				message: "wtᷥ",
			},
			want: true},
		{name: "¡ȸɯʚ",
			fields: fields{
				message: "¡ȸɯʚ",
			},
			want: false},
		{name: "¿ȸɯʚ",
			fields: fields{
				message: "¿ȸɯʚ",
			},
			want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &WTFSteroidChecker{
				message: tt.fields.message,
			}
			if got := w.ContainsWTF(); got != tt.want {
				t.Errorf("WTFSteroidChecker.ContainsWTF() = %v, want %v", got, tt.want)
			}
		})
	}

}

// TestWTFSteroidChecker_WTFUnicodeLibrary_Unique_Check check that all symbols in the library are unique
// and key ASCII symbol is not in library
func TestWTFSteroidChecker_WTFUnicodeLibrary_Unique_Check(t *testing.T) {
	w := WTFSteroidChecker{}
	unicodeLibrary := w.WTFUnicodeLibrary()
	if len(unicodeLibrary) == 0 {
		t.Errorf("Library is empty")
	}
	checkMap := make(map[string]struct{})
	for mainLetter, listOfUnicodes := range unicodeLibrary {
		for numInArray, unicodeSymbol := range listOfUnicodes {
			_, ok := checkMap[unicodeSymbol]
			if !ok {
				checkMap[unicodeSymbol] = struct{}{}
			} else {
				t.Errorf("Duplicate symbol is %s looks like %s and in %d position in array", unicodeSymbol, mainLetter, numInArray)
			}
			if unicodeSymbol == mainLetter {
				t.Errorf("Library should not have letter of a key letter %s in %d position in array", mainLetter, numInArray)
			}
		}
	}
}

// TestWTFSteroidChecker_WTFUnicodeDiacriticLibrary_Unique_Check check that all symbols in diacritic library are unique
// and key ASCII symbol is not in library
func TestWTFSteroidChecker_WTFUnicodeDiacriticLibrary_Unique_Check(t *testing.T) {
	w := WTFSteroidChecker{}
	unicodeLibrary := w.WTFUnicodeDiacriticLibrary()
	if len(unicodeLibrary) == 0 {
		t.Errorf("Library is empty")
	}
	checkMap := make(map[string]struct{})
	for mainLetter, listOfUnicodes := range unicodeLibrary {
		for numInArray, unicodeSymbol := range listOfUnicodes {
			_, ok := checkMap[unicodeSymbol]
			if !ok {
				checkMap[unicodeSymbol] = struct{}{}
			} else {
				t.Errorf("Duplicate symbol is %s looks like %s and in %d position in array", unicodeSymbol, mainLetter, numInArray)
			}
			if unicodeSymbol == mainLetter {
				t.Errorf("Library should not have letter of a key letter %s in %d position in array", mainLetter, numInArray)
			}
		}
	}
}

// TestWTFSteroidChecker_WTFUnicodeLibrary_Diacritic_Check library should not have diacritic symbols
// diacritic symbols remove separately
func TestWTFSteroidChecker_WTFUnicodeLibrary_Diacritic_Check(t *testing.T) {
	w := WTFSteroidChecker{}
	unicodeLibrary := w.WTFUnicodeLibrary()
	for mainLetter, listOfUnicodes := range unicodeLibrary {
		for numInArray, unicodeSymbol := range listOfUnicodes {
			trans := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
			withoutDiacritic, _, _ := transform.String(trans, unicodeSymbol)
			if unicodeSymbol != withoutDiacritic {
				t.Errorf("Should not be diacritic symbol in library %s looks like %s and in %d position in array", unicodeSymbol, mainLetter, numInArray)
			}
		}
	}
}

// TestWTFSteroidChecker_WTFUnicodeDiacriticLibrary_Diacritic_Check library should have only diacritic symbols
func TestWTFSteroidChecker_WTFUnicodeDiacriticLibrary_Diacritic_Check(t *testing.T) {
	w := WTFSteroidChecker{}
	unicodeLibrary := w.WTFUnicodeDiacriticLibrary()
	for mainLetter, listOfUnicodes := range unicodeLibrary {
		for numInArray, unicodeSymbol := range listOfUnicodes {
			trans := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
			withoutDiacritic, _, _ := transform.String(trans, unicodeSymbol)
			if unicodeSymbol == withoutDiacritic {
				t.Errorf("Should not be not diacritic in library %s looks like %s and in %d position in array", unicodeSymbol, mainLetter, numInArray)
			}
		}
	}
}
