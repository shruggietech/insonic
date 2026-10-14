@echo off
title insonic CLI
set "PATH=%INSONIC_PROGRAM_DIRECTORY%;%PATH%"
"%INSONIC_LAUNCH_CLI%" --help
echo.
echo Optional first-run setup: insonic setup location
echo Use insonic --help to inspect commands.
