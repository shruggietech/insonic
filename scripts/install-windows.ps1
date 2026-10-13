# SPDX-License-Identifier: Apache-2.0
param([Parameter(Mandatory=$true)][string]$Destination)
$ErrorActionPreference = 'Stop'
# Resolved directory handles make aliases and cleanup ownership independent of path spelling.
if (-not ('InsonicInstall.DirectoryLease' -as [type])) {
Add-Type -TypeDefinition @'
using System;
using System.ComponentModel;
using System.IO;
using System.Runtime.InteropServices;
using System.Text;
using Microsoft.Win32.SafeHandles;
namespace InsonicInstall {
 public sealed class DirectoryLease : IDisposable {
  [StructLayout(LayoutKind.Sequential)] struct FileInfo { public uint Attributes; public System.Runtime.InteropServices.ComTypes.FILETIME Creation,Access,Write; public uint Volume,SizeHigh,SizeLow,Links,IndexHigh,IndexLow; }
  [StructLayout(LayoutKind.Sequential)] struct UnicodeName { public ushort Length,MaximumLength; public IntPtr Buffer; }
  [StructLayout(LayoutKind.Sequential)] struct ObjectAttributes { public int Length; public IntPtr Root,Name; public uint Attributes; public IntPtr Security,Quality; }
  [StructLayout(LayoutKind.Sequential)] struct IoStatus { public IntPtr Status,Information; }
  [StructLayout(LayoutKind.Sequential)] struct RenameInfo { [MarshalAs(UnmanagedType.U1)] public bool Replace; public IntPtr Root; public uint Length; public char Name; }
  [DllImport("kernel32.dll",CharSet=CharSet.Unicode,SetLastError=true)] static extern SafeFileHandle CreateFile(string name,uint access,uint share,IntPtr security,uint creation,uint flags,IntPtr template);
  [DllImport("kernel32.dll",SetLastError=true)] static extern bool GetFileInformationByHandle(SafeFileHandle handle,out FileInfo info);
  [DllImport("kernel32.dll",CharSet=CharSet.Unicode,SetLastError=true)] static extern uint GetFinalPathNameByHandle(SafeFileHandle handle,StringBuilder value,uint length,uint flags);
  [DllImport("kernel32.dll",SetLastError=true)] static extern bool SetFileInformationByHandle(SafeFileHandle handle,int kind,IntPtr info,uint length);
  [DllImport("ntdll.dll")] static extern int NtCreateFile(out IntPtr handle,uint access,ref ObjectAttributes attributes,out IoStatus status,IntPtr allocation,uint flags,uint share,uint creation,uint options,IntPtr ea,uint eaLength);
  [DllImport("ntdll.dll")] static extern uint RtlNtStatusToDosError(int status);
  [DllImport("ntdll.dll")] static extern int NtSetInformationFile(SafeFileHandle handle,out IoStatus status,IntPtr info,uint length,int kind);
  SafeFileHandle handle; FileInfo identity;
  public string Path { get; private set; }
  DirectoryLease(SafeFileHandle value) {
   handle=value;
   if (!GetFileInformationByHandle(handle,out identity)) throw new Win32Exception(Marshal.GetLastWin32Error());
   var buffer=new StringBuilder(32768);uint size=GetFinalPathNameByHandle(handle,buffer,(uint)buffer.Capacity,0);
   if(size==0 || size>=buffer.Capacity) throw new Win32Exception(Marshal.GetLastWin32Error());
   string path=buffer.ToString();
   Path=path.StartsWith(@"\\?\UNC\",StringComparison.OrdinalIgnoreCase)?@"\\"+path.Substring(8):path.StartsWith(@"\\?\",StringComparison.Ordinal)?path.Substring(4):path;
  }
  public static DirectoryLease Open(string path) {
   var h=CreateFile(path,0x80,3,IntPtr.Zero,3,0x02000000,IntPtr.Zero);
   if(h.IsInvalid){h.Dispose();throw new Win32Exception(Marshal.GetLastWin32Error());}
   try {var lease=new DirectoryLease(h);if((lease.identity.Attributes & 0x10)==0)throw new IOException("Expected a directory.");return lease;}catch{h.Dispose();throw;}
  }
  DirectoryLease Relative(string name,bool create,bool deleteAccess=true) {
   if(string.IsNullOrEmpty(name)||name=="."||name==".."||name.IndexOfAny(new char[]{'\\','/','\0'})>=0)throw new IOException("Invalid relative installation name.");
   var raw=Marshal.StringToHGlobalUni(name);var uni=new UnicodeName{Length=(ushort)(name.Length*2),MaximumLength=(ushort)((name.Length+1)*2),Buffer=raw};
   var ptr=Marshal.AllocHGlobal(Marshal.SizeOf(typeof(UnicodeName)));
   try {
    Marshal.StructureToPtr(uni,ptr,false);var attrs=new ObjectAttributes{Length=Marshal.SizeOf(typeof(ObjectAttributes)),Root=handle.DangerousGetHandle(),Name=ptr,Attributes=0x40};
    IntPtr opened;IoStatus io;int status=NtCreateFile(out opened,deleteAccess?0x00110080u:0x00100080u,ref attrs,out io,IntPtr.Zero,create?0x10u:0u,3,create?2u:1u,0x00204020u|(create?1u:0u),IntPtr.Zero,0);
    if(status<0)throw new Win32Exception((int)RtlNtStatusToDosError(status));
    var h=new SafeFileHandle(opened,true);try{return new DirectoryLease(h);}catch{h.Dispose();throw;}
   } finally {Marshal.FreeHGlobal(ptr);Marshal.FreeHGlobal(raw);}
  }
  // FILE_CREATE atomically returns a handle to the new directory. No existing or replaced path acquires ownership.
  public DirectoryLease CreateAttempt(string name){return Relative(name,true);}
  public bool Contains(string selected) {
   string root=Path.TrimEnd('\\');return selected.Equals(root,StringComparison.OrdinalIgnoreCase)||selected.StartsWith(root+"\\",StringComparison.OrdinalIgnoreCase);
  }
  public void ValidateSourceTree() {
   foreach(string entry in Directory.GetFileSystemEntries(Path))using(var child=Relative(System.IO.Path.GetFileName(entry),false,false)) {
    if((child.identity.Attributes & 0x400)!=0)throw new IOException("The extracted source contains a reparse link; extract the complete official ZIP again.");
    if((child.identity.Attributes & 0x10)!=0)child.ValidateSourceTree();
   }
  }
  public void Promote(DirectoryLease parent,string name) {
   if(name.IndexOfAny(new char[]{'\\','/'})>=0)throw new IOException("Invalid installation name.");
   var bytes=Encoding.Unicode.GetBytes(name);int offset=Marshal.OffsetOf(typeof(RenameInfo),"Name").ToInt32();var info=Marshal.AllocHGlobal(offset+bytes.Length);
   try {
    for(int i=0;i<offset+bytes.Length;i++)Marshal.WriteByte(info,i,0);
    Marshal.WriteIntPtr(info,Marshal.OffsetOf(typeof(RenameInfo),"Root").ToInt32(),parent.handle.DangerousGetHandle());
    Marshal.WriteInt32(info,Marshal.OffsetOf(typeof(RenameInfo),"Length").ToInt32(),bytes.Length);Marshal.Copy(bytes,0,IntPtr.Add(info,offset),bytes.Length);
    // ReplaceIfExists=false. The owned handle stays locked throughout promotion.
    IoStatus io;int status=NtSetInformationFile(handle,out io,info,(uint)(offset+bytes.Length),10);
    if(status<0)throw new Win32Exception((int)RtlNtStatusToDosError(status));
   }finally{Marshal.FreeHGlobal(info);}
  }
  void MarkDeleted() {
   var info=Marshal.AllocHGlobal(4);try{Marshal.WriteInt32(info,1);if(!SetFileInformationByHandle(handle,4,info,4))throw new Win32Exception(Marshal.GetLastWin32Error());}finally{Marshal.FreeHGlobal(info);}
  }
  public void RemoveOwnedAttempt() {
   if((identity.Attributes & 0x10)!=0 && (identity.Attributes & 0x400)==0)foreach(string entry in Directory.GetFileSystemEntries(Path))using(var child=Relative(System.IO.Path.GetFileName(entry),false)){child.RemoveOwnedAttempt();}
   // Relative opens never follow reparse points; deletion affects the held object, not a substituted path.
   MarkDeleted();
  }
  public void Dispose(){if(handle!=null){handle.Dispose();handle=null;}}
 }
}
'@
}
$sourceLease = $null
$parentLease = $null
$attempt = $null
$completed = $false
try {
 $selected = [IO.Path]::GetFullPath($Destination).TrimEnd([IO.Path]::DirectorySeparatorChar)
 if ([IO.Directory]::Exists($selected) -or [IO.File]::Exists($selected)) { throw 'Choose a new installation directory.' }
 $parent = [IO.Path]::GetDirectoryName($selected)
 if (-not [IO.Directory]::Exists($parent)) { throw 'Choose a destination whose parent directory already exists.' }
 $sourceLease = [InsonicInstall.DirectoryLease]::Open($PSScriptRoot)
 $parentLease = [InsonicInstall.DirectoryLease]::Open($parent)
 $leaf = [IO.Path]::GetFileName($selected)
 $physicalTarget = [IO.Path]::Combine($parentLease.Path, $leaf)
 if ($sourceLease.Contains($physicalTarget)) { throw 'The destination must be outside the extracted source tree.' }
 if ([IO.Directory]::Exists($physicalTarget) -or [IO.File]::Exists($physicalTarget)) { throw 'Choose a new installation directory.' }
 $sourceLease.ValidateSourceTree()
 $attempt = $parentLease.CreateAttempt('.insonic-install-' + [Guid]::NewGuid().ToString('N'))
 Get-ChildItem -LiteralPath $sourceLease.Path -Force | Sort-Object Name | ForEach-Object { Copy-Item -LiteralPath $_.FullName -Destination $attempt.Path -Recurse }
 $attempt.Promote($parentLease, $leaf)
 $completed = $true
 Write-Output ('Installed insonic in ' + $selected + '. CLIENT_INSTRUCTIONS')
} catch {
 $original = $_
 if ($attempt -and -not $completed) {
  try { $attempt.RemoveOwnedAttempt() } catch { Write-Warning ('The owned partial attempt could not be removed: ' + $_.Exception.Message) }
 }
 throw $original
} finally {
 if ($attempt) { $attempt.Dispose() }
 if ($parentLease) { $parentLease.Dispose() }
 if ($sourceLease) { $sourceLease.Dispose() }
}
