#ifndef YOINK_CLIPBOARD_DARWIN_H
#define YOINK_CLIPBOARD_DARWIN_H

// yoink_change_count returns NSPasteboard.changeCount.
long yoink_change_count(void);

// yoink_read returns a malloc'd UTF-8 copy of the clipboard string (caller
// frees), or NULL when there is no string content. *concealed is set to 1 when
// the pasteboard carries a concealed/transient marker.
char *yoink_read(int *concealed);

// yoink_write replaces the clipboard contents with text. Returns 0 on success.
int yoink_write(const char *text);

#endif
