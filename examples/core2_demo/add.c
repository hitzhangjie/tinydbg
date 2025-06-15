#include <stdint.h>
#include <signal.h>

int64_t add(int64_t a, int64_t b)
{
    return a + b;
}

// SIGSEG will be captured by go runtime, to convert it to SIGABRT (6).
// The default behavior is let kernel dump the core file.
// But go runtime handle this SIGABRT, so no coredump will be generated.
//
// As go blogs, add GOTRACEBACK=crash will create the coredump, but
// there must be something wrong, no coredump will be generated.
// Maybe it's relevant with the cgocall.
//
// OK, let's make it simple, we uses SIGBUS to trigger a coredump.
int64_t bad_add(int64_t a, int64_t b)
{
    raise(SIGBUS);
    return 0;
}