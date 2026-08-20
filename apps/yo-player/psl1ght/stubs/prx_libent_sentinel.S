/*
 * Official SDK executables keep an eight-byte zero entry in the process
 * library-export range even when the application exports no PRX library.
 * GameOS uses this range while binding dependencies between loaded SPRXs.
 */

   .section ".lib.ent","a"
   .align 3
   .quad 0
