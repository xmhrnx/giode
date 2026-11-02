package giode

import (
	"context"
	"errors"

	"gioui.org/app"

	"github.com/xmhrnx/giode/internal/mobile"
)

func (a *App) pickPhoto(ctx context.Context, opts PhotoOptions) (PickedFile, error) {
	var ch <-chan mobile.Photo
	err := a.onAndroidMain(ctx, func(view uintptr) error {
		var err error
		ch, err = mobile.PickPhoto(app.JavaVM(), app.AppContext(), view, opts.MaxSide, opts.Source == Camera)
		return err
	})
	if err != nil {
		return PickedFile{}, err
	}
	select {
	case p := <-ch:
		switch {
		case errors.Is(p.Err, mobile.ErrCanceled):
			return PickedFile{}, ErrCanceled
		case p.Err != nil:
			return PickedFile{}, p.Err
		}
		return jpegFile(p.Name, p.JPEG), nil
	case <-ctx.Done():
		return PickedFile{}, ctx.Err()
	}
}

func (a *App) pickPhotos(ctx context.Context, opts PhotoOptions, limit int) ([]PickedFile, error) {
	var ch <-chan mobile.Photos
	err := a.onAndroidMain(ctx, func(view uintptr) error {
		var err error
		ch, err = mobile.PickPhotos(app.JavaVM(), app.AppContext(), view, opts.MaxSide, limit)
		return err
	})
	if err != nil {
		return nil, err
	}
	select {
	case ps := <-ch:
		switch {
		case errors.Is(ps.Err, mobile.ErrCanceled):
			return nil, ErrCanceled
		case ps.Err != nil:
			return nil, ps.Err
		}
		files := make([]PickedFile, len(ps.Photos))
		for i, p := range ps.Photos {
			files[i] = jpegFile(p.Name, p.JPEG)
		}
		return files, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
