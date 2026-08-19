#pragma once

#include <stdint.h>
#include <time.h>
#include <sys/systime.h>

typedef struct CellRtcDateTime {
   uint16_t year, month, day, hour, minute, second;
   uint32_t microsecond;
} CellRtcDateTime;

typedef struct CellRtcTick { uint64_t tick; } CellRtcTick;

static inline int pslRtcFromTime(time_t value, uint32_t microseconds,
                                 CellRtcDateTime *out)
{
   struct tm localValue;
   if (!out || !localtime_r(&value, &localValue)) return -1;
   out->year = (uint16_t)(localValue.tm_year + 1900);
   out->month = (uint16_t)(localValue.tm_mon + 1);
   out->day = (uint16_t)localValue.tm_mday;
   out->hour = (uint16_t)localValue.tm_hour;
   out->minute = (uint16_t)localValue.tm_min;
   out->second = (uint16_t)localValue.tm_sec;
   out->microsecond = microseconds;
   return 0;
}

static inline int cellRtcGetCurrentClockLocalTime(CellRtcDateTime *out)
{
   uint64_t seconds = 0, nanoseconds = 0;
   if (sysGetCurrentTime(&seconds, &nanoseconds) != 0) return -1;
   return pslRtcFromTime((time_t)seconds, (uint32_t)(nanoseconds / 1000), out);
}

static inline int cellRtcSetTime_t(CellRtcDateTime *out, uint64_t seconds)
{
   return pslRtcFromTime((time_t)seconds, 0, out);
}

static inline int cellRtcGetTick(const CellRtcDateTime *value, CellRtcTick *tick)
{
   if (!value || !tick) return -1;
   struct tm parts;
   parts.tm_sec = value->second; parts.tm_min = value->minute;
   parts.tm_hour = value->hour; parts.tm_mday = value->day;
   parts.tm_mon = value->month - 1; parts.tm_year = value->year - 1900;
   parts.tm_wday = 0; parts.tm_yday = 0; parts.tm_isdst = -1;
   time_t seconds = mktime(&parts);
   if (seconds == (time_t)-1) return -1;
   tick->tick = (uint64_t)seconds;
   return 0;
}

static inline int cellRtcConvertUtcToLocalTime(const CellRtcTick *utc,
                                               CellRtcTick *local)
{
   if (!utc || !local) return -1;
   local->tick = utc->tick;
   return 0;
}

static inline int cellRtcSetTick(CellRtcDateTime *out, const CellRtcTick *tick)
{
   return tick ? pslRtcFromTime((time_t)tick->tick, 0, out) : -1;
}
