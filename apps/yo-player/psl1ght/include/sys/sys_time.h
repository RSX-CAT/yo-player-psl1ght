#pragma once

#include <stdint.h>
#include <sys/systime.h>

typedef uint64_t sys_time_sec_t;
typedef uint64_t sys_time_nsec_t;

static inline int sys_time_get_current_time(sys_time_sec_t *seconds,
                                            sys_time_nsec_t *nanoseconds)
{
   return sysGetCurrentTime(seconds, nanoseconds);
}

static inline system_time_t sys_time_get_system_time(void)
{
   uint64_t seconds = 0, nanoseconds = 0;
   if (sysGetCurrentTime(&seconds, &nanoseconds) != 0) return 0;
   return seconds * 1000000ull + nanoseconds / 1000ull;
}
