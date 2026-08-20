.SUFFIXES:

ifeq ($(strip $(PSL1GHT)),)
$(error "Set PSL1GHT to the installed PSL1GHT directory")
endif

include $(PSL1GHT)/ppu_rules

# GNU make supplies a built-in LD=ld, so PSL1GHT's conditional LD assignment
# does not replace it.  Link through the PPU GCC driver selected by ppu_rules.
LD := $(CC)

TARGET  := yo-player-psl1ght
BUILD   := build/psl1ght
SOURCES := src src/screens src/extractors psl1ght/stubs
INCLUDES := include ../../libs/simple-lib-app/include ../../libs/simple-lib-av/include \
            ../../libs/simple-lib-https/include ../../libs/simple-lib-core/include psl1ght/include

CFILES := main.c home.c play.c search.c video-grid.c storage.c stream-select.c \
          downloads.c video-export-psl1ght.c extractor.c youtube.c sponsorblock.c \
          subtitles.c chapters.c settings.c theme.c
FONT_STUB_OFILES := cellfont_stub.o cellfont_libexport.o \
                    cellfontft_stub.o cellfontft_libexport.o

TITLE     := Yo! Player (PSL1GHT)
APPID     := YOPSL0001
CONTENTID := UP0001-$(APPID)_00-0000000000000000
ICON0     := $(CURDIR)/ICON0.PNG
SFOXML    := $(CURDIR)/psl1ght/PARAM.SFO.xml

PROJECT_DIR := $(patsubst %/,%,$(dir $(abspath $(firstword $(MAKEFILE_LIST)))))
ROOT        := $(abspath $(PROJECT_DIR)/../..)
CORE_LIB   := $(ROOT)/libs/simple-lib-core/build/psl1ght/libsimple-lib-core.a
HTTPS_LIB  := $(ROOT)/libs/simple-lib-https/build/psl1ght/libsimple-lib-https.a
AV_LIB     := $(ROOT)/libs/simple-lib-av/build/psl1ght/libsimple-lib-av.a
APP_LIB    := $(ROOT)/libs/simple-lib-app/build/psl1ght/libsimple-lib-app.a
PORTLIBS   := $(PS3DEV)/portlibs/ppu

CFLAGS = -O2 -Wall -Wextra -std=gnu11 -mcpu=cell $(MACHDEP) $(INCLUDE) \
          -DPSL1GHT -DNDEBUG -include psl1ght_compat.h \
          -DBUILD_NAME=\"yo-player-psl1ght\" -DBUILD_STAMP=\"$(shell git -C $(ROOT) rev-parse --short HEAD 2>/dev/null || echo unknown)\"
ASFLAGS = $(MACHDEP) -mregnames -mcpu=cell -Wa,-mcell $(INCLUDE)
LDFLAGS = $(MACHDEP) -Wl,--gc-sections -Wl,-Map,$(notdir $@).map
PROJECT_LIBDIRS := $(ROOT)/libs/simple-lib-app/build/psl1ght \
                   $(ROOT)/libs/simple-lib-av/build/psl1ght \
                   $(ROOT)/libs/simple-lib-https/build/psl1ght \
                   $(ROOT)/libs/simple-lib-core/build/psl1ght
LIBS := -lsimple-lib-app -lsimple-lib-av -lsimple-lib-https -lsimple-lib-core \
        -lfaad -lvdec -laudio -lrsx -lgcm_sys -lio -lsysutil \
        -lpngdec -ljpgdec -lsysmodule -lnet -lsysfs -lrt -llv2 -lm

ifneq ($(notdir $(BUILD)),$(notdir $(CURDIR)))
export OUTPUT := $(CURDIR)/$(TARGET)
export VPATH := $(foreach dir,$(SOURCES),$(CURDIR)/$(dir))
export DEPSDIR := $(CURDIR)/$(BUILD)
export BUILDDIR := $(CURDIR)/$(BUILD)
# sprxlinker derives each import count from the distance between consecutive
# FNID marker symbols.  Keep every assembly marker immediately before its C
# table object; grouping all .S files separately would make both counts zero.
export OFILES := $(FONT_STUB_OFILES) $(CFILES:.c=.o)
export INCLUDE := $(foreach dir,$(INCLUDES),-I$(CURDIR)/$(dir)) \
                  -I$(PORTLIBS)/include $(LIBPSL1GHT_INC) -I$(CURDIR)/$(BUILD)
export LIBPATHS := $(foreach dir,$(PROJECT_LIBDIRS),-L$(dir)) -L$(PORTLIBS)/lib $(LIBPSL1GHT_LIB)

.PHONY: all libraries $(CORE_LIB) $(HTTPS_LIB) $(AV_LIB) $(APP_LIB) $(BUILD) clean run pkg
all: $(BUILD)

libraries: $(CORE_LIB) $(HTTPS_LIB) $(AV_LIB) $(APP_LIB)

$(CORE_LIB):
	@$(MAKE) --no-print-directory -C $(ROOT)/libs/simple-lib-core -f Makefile.psl1ght

$(HTTPS_LIB):
	@$(MAKE) --no-print-directory -C $(ROOT)/libs/simple-lib-https -f Makefile.psl1ght

$(AV_LIB):
	@$(MAKE) --no-print-directory -C $(ROOT)/libs/simple-lib-av -f Makefile.psl1ght

$(APP_LIB):
	@$(MAKE) --no-print-directory -C $(ROOT)/libs/simple-lib-app -f Makefile.psl1ght

$(BUILD): libraries
	@[ -d $@ ] || mkdir -p $@
	@$(MAKE) --no-print-directory -C $@ -f $(CURDIR)/Makefile

clean:
	@$(MAKE) --no-print-directory -C $(ROOT)/libs/simple-lib-core -f Makefile.psl1ght clean
	@$(MAKE) --no-print-directory -C $(ROOT)/libs/simple-lib-https -f Makefile.psl1ght clean
	@$(MAKE) --no-print-directory -C $(ROOT)/libs/simple-lib-av -f Makefile.psl1ght clean
	@$(MAKE) --no-print-directory -C $(ROOT)/libs/simple-lib-app -f Makefile.psl1ght clean
	@rm -rf $(BUILD) $(OUTPUT).elf $(OUTPUT).self $(OUTPUT).fake.self $(OUTPUT).pkg $(OUTPUT).gnpdrm.pkg

run: $(BUILD)
	@$(PS3LOADAPP) $(OUTPUT).self

# Keep package creation behind the completed recursive build even under -j.
# The conventional `pkg: $(BUILD) $(OUTPUT).pkg` form permits both prerequisites
# to run concurrently and can start a second object build in this directory.
pkg: $(BUILD)
	@rm -f $(OUTPUT).pkg $(OUTPUT).gnpdrm.pkg
	@$(MAKE) --no-print-directory $(OUTPUT).pkg
else
DEPENDS := $(OFILES:.o=.d)

# PRX import headers contain 32-bit guest pointers.  Match PSL1GHT's own
# libexport build rule: emit the C initializer with -m32, then assemble it.
%_libexport.o: %_libexport.c
	@$(CC) $(DEPSOPT) -S -m32 $(INCLUDE) $< -o $(@:.o=.generated.S)
	@$(CC) $(DEPSOPT) -c $(@:.o=.generated.S) -o $@

$(OUTPUT).self: $(OUTPUT).elf
	@echo CEX self ... $(notdir $@)
	@mkdir -p $(BUILDDIR)
	@$(STRIP) $< -o $(BUILDDIR)/$(notdir $<)
	@$(SPRX) $(BUILDDIR)/$(notdir $<)
	@python3 $(CURDIR)/psl1ght/set_prx_sdk_version.py \
		$(BUILDDIR)/$(notdir $<) --sdk 0x00475001
	@$(SELF) $(BUILDDIR)/$(notdir $<) $@
	@$(FSELF) $(BUILDDIR)/$(notdir $<) $(basename $@).fake.self
$(OUTPUT).elf: $(OFILES) $(APP_LIB) $(AV_LIB) $(HTTPS_LIB) $(CORE_LIB)
-include $(DEPENDS)
endif
