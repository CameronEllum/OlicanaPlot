@echo off
REM Initialize Visual Studio build environment
call "C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Auxiliary\Build\vcvarsall.bat" x64 >nul 2>&1

REM Build the plugin
cl /EHsc /O2 /std:c++20 /Fe:random_walk_generator.exe main.cpp user32.lib gdi32.lib
