#include <stdint.h>

void *giodeAudioOpen(int capture, int render, int voice, int speaker, double rate, int channels, const char *device, char **err);
void giodeAudioSetRender(void *ref, int on);
void giodeAudioClose(void *ref);
int giodeAudioRead(void *ref, int16_t *dst, int n, int ms);
int giodeAudioWrite(void *ref, const int16_t *src, int n);
int giodeAudioPermission(void);
char *giodeAudioDevices(int input);
void giodeAudioPreferInput(const char *uid);
