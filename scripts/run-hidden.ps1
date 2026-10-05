# SPDX-License-Identifier: Apache-2.0
# Maintainer launcher: literal arguments, redirected I/O, no console window.
param(
    [Parameter(Mandatory = $true)][string]$Executable,
    [Parameter(ValueFromRemainingArguments = $true)][string[]]$Arguments
)
$ErrorActionPreference = 'Stop'
$start = [System.Diagnostics.ProcessStartInfo]::new()
$start.FileName = $Executable
$start.UseShellExecute = $false
$start.CreateNoWindow = $true
$start.RedirectStandardInput = $true
$start.RedirectStandardOutput = $true
$start.RedirectStandardError = $true
foreach ($argument in $Arguments) { $start.ArgumentList.Add($argument) }
$start.Environment['GIT_TERMINAL_PROMPT'] = '0'
$start.Environment['CI'] = 'true'
$child = [System.Diagnostics.Process]::new()
$child.StartInfo = $start
try {
    [void]$child.Start()
    $child.StandardInput.Close()
    $stdout = $child.StandardOutput.ReadToEndAsync()
    $stderr = $child.StandardError.ReadToEndAsync()
    $child.WaitForExit()
    [Console]::Out.Write($stdout.GetAwaiter().GetResult())
    [Console]::Error.Write($stderr.GetAwaiter().GetResult())
    $result = $child.ExitCode
} finally { $child.Dispose() }
exit $result
