#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>
#include "clipboard_darwin.h"

// Pasteboard markers used by password managers and similar tools to flag
// secrets that clipboard history should not retain.
static NSString *const kConcealedType = @"org.nspasteboard.ConcealedType";
static NSString *const kTransientType = @"org.nspasteboard.TransientType";

long yoink_change_count(void) {
    @autoreleasepool {
        return (long)[[NSPasteboard generalPasteboard] changeCount];
    }
}

char *yoink_read(int *concealed) {
    @autoreleasepool {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];

        *concealed = 0;
        for (NSString *t in [pb types]) {
            if ([t isEqualToString:kConcealedType] ||
                [t isEqualToString:kTransientType]) {
                *concealed = 1;
                break;
            }
        }

        NSString *s = [pb stringForType:NSPasteboardTypeString];
        if (s == nil) {
            return NULL;
        }
        const char *utf8 = [s UTF8String];
        if (utf8 == NULL) {
            return NULL;
        }
        return strdup(utf8);
    }
}

int yoink_write(const char *text) {
    @autoreleasepool {
        NSString *s = [NSString stringWithUTF8String:text];
        if (s == nil) {
            return 1;
        }
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        [pb clearContents];
        return [pb setString:s forType:NSPasteboardTypeString] ? 0 : 1;
    }
}
