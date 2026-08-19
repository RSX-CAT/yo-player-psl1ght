#pragma once

#include <string.h>
#include <lv2/mutex.h>

typedef sys_lwmutex_attr_t sys_lwmutex_attribute_t;

#define SYS_SYNC_RECURSIVE SYS_LWMUTEX_ATTR_RECURSIVE

#define sys_lwmutex_attribute_initialize(value) do { \
   memset(&(value), 0, sizeof(value)); \
   (value).attr_protocol = SYS_LWMUTEX_PROTOCOL_PRIO; \
   (value).attr_recursive = SYS_LWMUTEX_ATTR_NOT_RECURSIVE; \
} while (0)

#define sys_lwmutex_create  sysLwMutexCreate
#define sys_lwmutex_destroy sysLwMutexDestroy
#define sys_lwmutex_lock    sysLwMutexLock
#define sys_lwmutex_unlock  sysLwMutexUnlock
