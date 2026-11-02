package pdfpages

import (
	"github.com/xmhrnx/giode/android"
)

//go:generate javac --release 8 -classpath $ANDROID_HOME/platforms/android-34/android.jar -d $TEMP/giode_pdfpages/classes pdfpages_android.java
//go:generate jar cf pdfpages_android.jar -C $TEMP/giode_pdfpages/classes .

const (
	supported = true
	class     = "org/giode/pdfpages/pdfpages_android"
)

type platformDoc struct{ id int }

func open(path string) (platformDoc, int, error) {
	id, err := android.CallInt(class, "open", "(Ljava/lang/String;)I", path)
	if err != nil {
		return platformDoc{}, 0, err
	}
	n, err := android.CallInt(class, "pageCount", "(I)I", id)
	if err != nil {
		android.CallVoid(class, "close", "(I)V", id)
		return platformDoc{}, 0, err
	}
	return platformDoc{id: id}, n, nil
}

func (d platformDoc) pageSize(i int) (float64, float64, error) {
	v, err := android.CallInt(class, "pageSize", "(II)I", d.id, i)
	if err != nil {
		return 0, 0, err
	}
	return float64(uint32(v) >> 16), float64(uint32(v) & 0xffff), nil
}

func (d platformDoc) render(i, w, h int) ([]byte, error) {
	return android.CallBytes(class, "render", "(IIII)[B", d.id, i, w, h)
}

func (d platformDoc) close() error {
	return android.CallVoid(class, "close", "(I)V", d.id)
}
