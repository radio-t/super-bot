package bot

import (
	"testing"
)

// TestWTFSteroidChecker_Contains check that all possible messages can be recognized correctly
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

// Test that all symbols in the library are unique
func TestWTFSteroidChecker_WTFUnicodeLibrary_Unique_Check(t *testing.T) {
	w := WTFSteroidChecker{}
	unicodeLibrary := w.WTFUnicodeLibrary()
	if len(unicodeLibrary) <= 0 {
		t.Errorf("Library is empty")
	}
	checkMap := make(map[string]struct{})
	for _, listOfUnicodes := range unicodeLibrary {
		for _, unicodeSymbol := range listOfUnicodes {
			_, ok := checkMap[unicodeSymbol]
			if !ok {
				checkMap[unicodeSymbol] = struct{}{}
			} else {
				t.Errorf("Duplicate symbol %s", unicodeSymbol)
			}
		}
	}
}
