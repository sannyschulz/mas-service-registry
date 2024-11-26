Build instructions for the storage module.

requires: sqlite3 

On Windows:

you need to install gcc first. You can use the mingw-w64 package from the MSYS2 project.

download and install MSYS2 from https://www.msys2.org/
msys2-base-x86_64-*.tar.zst: Same as .sfx.exe but as an ZSTD archive
unpack
move folder to C:\msys64
run msys2_shell.cmd
open shell ucrt.exe
install gcc:
pacman -S mingw-w64-ucrt-x86_64-gcc
pacman -S --needed base-devel mingw-w64-ucrt-x86_64-toolchain
set PATH to C:\msys64\ucrt64\bin

install c/c++ extension for vscode (optional)

install go-sqlite3:

go install github.com/mattn/go-sqlite3

after that go-sqlite3 should work
