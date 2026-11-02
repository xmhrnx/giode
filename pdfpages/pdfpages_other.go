//go:build !android && !darwin

package pdfpages

import "github.com/xmhrnx/giode"

const supported = false

type platformDoc struct{}

func open(string) (platformDoc, int, error) { return platformDoc{}, 0, giode.ErrNotSupported }

func (platformDoc) pageSize(int) (float64, float64, error) { return 0, 0, giode.ErrNotSupported }
func (platformDoc) render(int, int, int) ([]byte, error)   { return nil, giode.ErrNotSupported }
func (platformDoc) close() error                           { return nil }
