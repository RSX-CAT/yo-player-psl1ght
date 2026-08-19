#pragma once

#include <stdint.h>
#include <sys/thread.h>
#include <lv2/thread.h>

#define SYS_PPU_THREAD_CREATE_JOINABLE THREAD_JOINABLE

static inline s32 pslPpuThreadCreate(sys_ppu_thread_t *thread,
                                     void (*entry)(uint64_t), uint64_t arg,
                                     s32 priority, uint64_t stackSize,
                                     uint64_t flags, const char *name)
{
   return sysThreadCreate(thread, (void (*)(void *))entry, (void *)(uintptr_t)arg,
                          priority, stackSize, flags, (char *)name);
}

#define sys_ppu_thread_create pslPpuThreadCreate
#define sys_ppu_thread_join   sysThreadJoin
#define sys_ppu_thread_exit   sysThreadExit
#define sys_ppu_thread_yield  sysThreadYield
