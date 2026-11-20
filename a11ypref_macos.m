//go:build darwin && !ios

@import AppKit;

int giodeIncreaseContrast(void) {
	return [[NSWorkspace sharedWorkspace] accessibilityDisplayShouldIncreaseContrast] ? 1 : 0;
}

// macOS has no bold text setting, and no system text size.
int giodeBoldText(void) {
	return -1;
}

double giodeTextScale(void) {
	return 0;
}
