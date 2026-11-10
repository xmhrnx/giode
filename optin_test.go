package giode

import (
	"context"
	"errors"
	"testing"
)

// Without the dictate package, Dictate says what's missing.
func TestDictateNeedsPackage(t *testing.T) {
	var a *App
	if _, err := a.Dictate(context.Background(), ""); !errors.Is(err, ErrNotSupported) {
		t.Errorf("err = %v, want ErrNotSupported", err)
	}
}

// Without their packages, taking a photo and saving to the gallery say
// what's missing.
func TestPhotoFeaturesNeedPackages(t *testing.T) {
	var a *App
	if _, err := a.PickPhoto(context.Background(), PhotoOptions{Source: Camera}); !errors.Is(err, ErrNotSupported) {
		t.Errorf("PickPhoto(Camera) err = %v, want ErrNotSupported", err)
	}
	if err := a.SaveToPhotos(context.Background(), "dictate_test.go", ""); !errors.Is(err, ErrNotSupported) {
		t.Errorf("SaveToPhotos err = %v, want ErrNotSupported", err)
	}
}
