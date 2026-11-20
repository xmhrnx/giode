package filedrag

import (
	"reflect"
	"testing"
)

func TestParseURIList(t *testing.T) {
	list := "# a comment\r\nfile:///home/me/My%20Video.mkv\r\nfile://localhost/tmp/a.txt\r\nhttps://example.com/x\r\nfile://otherhost/share/b\r\n\r\n"
	got := ParseURIList(list)
	want := []string{"/home/me/My Video.mkv", "/tmp/a.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseURIList = %q, want %q", got, want)
	}
}
