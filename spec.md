# AVFS Specification

Module: `github.com/avfs/avfs` · Go >= 1.26 · License: Apache-2.0

**AVFS** (Another Virtual File System) is a Go library that provides an abstraction
layer emulating the behaviour of a file system. It is inspired by
[Afero](https://github.com/spf13/afero) and the Go standard library (`os`,
`io/fs`, `path/filepath`).

The library provides:

- a set of **constants**, **interfaces** and **types** shared by all file systems,
- a reusable **test suite** applicable to any conforming file system,
- one package per **file system** implementation,
- a minimal **identity manager** abstraction (users and groups) allowing
  `Chown`/`Lchown` and permission checks to be tested.

---

## 1. Goals and non-goals

### Goals

1. Offer a single `VFS` abstraction that mirrors the most common `os` and
   `path/filepath` functions, so that swapping the host file system for an
   emulated one is a matter of replacing the package-level variable.
2. Emulate the semantics of a foreign operating system (Linux or Windows)
   independently of the host OS, including path separators, volumes and error
   values.
3. Allow identity (user/group) management to be plugged in, so ownership and
   permission logic is testable.
4. Provide a single, reusable conformance test suite so every implementation is
   validated against identical expectations.

### Non-goals

- Being a drop-in replacement for all of `os` (only the commonly used subset is
  modelled).
- Providing production-grade identity management (the bundled identity managers
  are explicitly test/dev oriented).
- Performance: in-memory implementations trade speed for isolation and
  reproducibility.

---

## 2. Package layout

| Path | Role |
|------|------|
| `avfs` (root) | Core contracts: `VFS`, `File`, `Features`, `IdmMgr`, error model, path helpers, generic free functions |
| `vfs/memfs` | Full-featured in-memory file system (reference implementation) |
| `vfs/orefafs` | Simplified Afero-like in-memory file system |
| `vfs/osfs` | Native OS file system, thin wrapper over `os`/`path/filepath` |
| `vfs/ostestfs` | Native OS file system + OS identity manager + `Chroot` |
| `vfs/basepathfs` | Decorator restricting all operations to a base path (chroot-like) |
| `vfs/rofs` | Read-only decorator |
| `vfs/failfs` | Fault-injection decorator |
| `idm/memidm` | In-memory identity manager |
| `idm/osidm` | Identity manager backed by OS commands (testing only) |
| `test` | Generic conformance test suite, benchmarks and fixtures |
| `test/testbuild` | Cross-compilation smoke program referencing every OS-specific symbol |
| `mage` | Build tooling (gox cross-compilation, lint, release) |

---

## 3. Core contracts

### 3.0 Embeddable mixins

Each small capability interface has a corresponding embeddable struct providing
a default implementation together with the state it needs. A file system
embeds the mixins rather than reimplementing them.

| Mixin | Implements | State held | Composition |
|-------|-----------|------------|-------------|
| `FeaturesMixin` | `Featurer` | feature bitmask | — |
| `UMaskMixin` | `UMasker` | `atomic.Uint32` | — |
| `OSTypeMixin` | `OSTyper` | OS type, separator, default modes | — |
| `IdmMixin` | `IdmProvider` | identity manager | — |
| `PathMixin` | `VFSPath` | — | `FeaturesMixin`, `UMaskMixin`, `OSTypeMixin` |
| `UserDirMixin` | `VFSUserDir` | current user, cwd, home, temp dirs | `IdmMixin`, `PathMixin` |

Because `PathMixin` transitively provides the other three, embedding
`UserDirMixin` alone satisfies `VFSBase` — this is what `memfs` and `orefafs`
do. Decorators (`RoFS`, `FailFS`, `BasePathFS`) embed only `FeaturesMixin`,
since they delegate the rest to a wrapped base file system.

A mixin is an implementation detail of a file system type: it is meant to be
embedded and its methods promoted, not selected by name.

### 3.1 `VFS` and `VFSBase`

`VFSBase` is the union of the small capability interfaces:

| Embedded interface | Methods |
|--------------------|---------|
| `Featurer` | `Features() Features`, `HasFeature(Features) bool` |
| `IdmProvider` | `Idm() IdmMgr`, `SetIdm(IdmMgr) error` |
| `Namer` | `Name() string` |
| `OSTyper` | `OSType() OSType` |
| `Typer` | `Type() string` |
| `UMasker` | `SetUMask(fs.FileMode) error`, `UMask() fs.FileMode` |
| `VFSPath` | path manipulation utilities (see §3.4) |
| `VFSUserDir` | `Abs`, `Getwd`, `SetUser`, `SetUserByName`, `TempDir`, `User`, `UserHomeDir` |

`VFSBase` additionally declares the operating-system operations:
`Base`, `Chdir`, `Chmod`, `Chown`, `Chtimes`, `Create`, `CreateTemp`,
`EvalSymlinks`, `Glob`, `UserHomeDir`, `Idm`, `Lchown`, `Link`, `Lstat`,
`Mkdir`, `MkdirAll`, `MkdirTemp`, `OpenFile`, `ReadDir`, `ReadFile`,
`Readlink`, `Remove`, `RemoveAll`, `Rename`, `SameFile`, `Stat`, `Symlink`,
`Truncate`, `WalkDir`, `WriteFile`.

`VFS` adds `Open(name) (File, error)` and `Sub(dir) (VFS, error)`.

`IOFS` is the `io/fs` projection: `VFSBase` plus `fs.FS`, `fs.GlobFS`,
`fs.ReadDirFS`, `fs.ReadFileFS`, `fs.StatFS`, `fs.SubFS`. It lets a VFS be used
directly where an `fs.FS` is expected.

Method naming is deliberately identical to the `os`/`path/filepath`
counterpart, and error types mirror the standard library: `*fs.PathError` for
path operations, `*os.LinkError` for `Link`/`Rename`/`Symlink`.

### 3.2 `File`

`File` embeds `fs.File`, `fs.ReadDirFile`, `io.Reader`, `io.ReaderAt`,
`io.StringWriter`, `io.Writer`, `io.WriterAt`, `io.WriteSeeker` and adds
`Chdir`, `Chmod`, `Chown`, `Fd`, `Name`, `Readdirnames`, `Sync`, `Truncate`.

### 3.3 Optional capabilities

Two small interfaces allow optional behaviour:

- `Cloner` — `Clone() VFS`, a shallow copy of the file system.
- `ChRooter` — `Chroot(path string) error`, changes the file system root
  (requires privileges; `ErrPermDenied` otherwise).

### 3.4 `VFSPath`

OS-aware replacements for `path/filepath`, parameterised by the *emulated*
`OSType`, not the host: `Base`, `Clean`, `Dir`, `FromSlash`, `IsAbs`,
`IsPathSeparator`, `Join`, `Match`, `PathSeparator`, `Rel`, `Split`, `ToSlash`,
`ToSysStat`, `VolumeName`, `VolumeNameLen`.

`VolumeManager` (`VolumeAdd`, `VolumeDelete`, `VolumeList`) is implemented by
file systems emulating Windows volumes.

`PathMixin` has **two implementations of the same interface**, selected at
build time by the `avfs_setostype` tag (see §3.6):

| Build | Implementation | Semantics |
|-------|----------------|-----------|
| default | `vfs_path_ostype_off.go`, delegating to `path/filepath` | the host OS |
| `avfs_setostype` | `vfs_path_ostype_on.go`, `path/filepath` logic parameterised by `PathSeparator`/`VolumeNameLen` | the emulated `OSType` |

Both provide the *same* method set, so `PathMixin` satisfies `VFSPath` in either
configuration; no method may be dropped or renamed by the tag. `ToSysStat` and
`VolumeManager` are tag-independent.

### 3.5 Feature flags

`Features` is a `uint64` bitmask:

| Flag | Meaning |
|------|---------|
| `FeatHardlink` | `Link` is supported |
| `FeatIdentityMgr` | An identity manager exists; multiple users supported |
| `FeatSetOSType` | The emulated OS can be changed at runtime (build tag `avfs_setostype`, see §3.6) |
| `FeatReadOnly` | The file system rejects every mutation |
| `FeatReadOnlyIdm` | The identity manager rejects mutations |
| `FeatRealFS` | The file system is a real one, not emulated |
| `FeatSubFS` | `Sub` is supported |
| `FeatSymlink` | `Symlink`/`Readlink`/`EvalSymlinks` are supported |

`FeaturesMixin` is an embeddable implementation (`Features`, `HasFeature`,
`SetFeatures`). `BuildFeatures()` returns the flags enabled by the build
(`FeatSetOSType` only under `avfs_setostype`).

Feature flags are normative: the test suite uses them to decide which tests to
run and which operations must succeed. `FeatSetOSType` is additionally bound to
the build configuration — an implementation advertises it only when the binary
was built with `avfs_setostype`, and then only if its `SetOSType` honours the
derived-state rules of §3.6.

### 3.6 Operating system emulation

`OSType` is one of `OsUnknown`, `OsLinux`, `OsWindows`, `OsDarwin`.
`OSTypeMixin` (embeddable) provides `OSType`, `SetOSType`, `PathSeparator`,
`DirMode`, `FileMode`. `CurrentOSType()` reports the host OS.

#### The two build configurations

Emulating a foreign OS requires that path handling be parameterised by the
*emulated* OS instead of the host, which `path/filepath` cannot do (its
separator is fixed per `GOOS` at compile time). Rather than always paying for
that, the capability is a build-time choice:

| | default (`!avfs_setostype`) | `avfs_setostype` |
|---|---|---|
| `VFSPath` methods | delegate to `path/filepath` | OS-parameterised (`vfs_path_ostype_on.go`) |
| Path semantics | host OS | emulated `OSType`, re-read on every call |
| `buildFeatSetOSType` / `BuildFeatures()` | `0` | `FeatSetOSType` |
| `Features()` of an emulated FS | no `FeatSetOSType` | `FeatSetOSType` |
| `SetOSType(foreignOS)` | `ErrSetOSType` | accepted |
| `SetOSType(OsUnknown)` / `SetOSType(CurrentOSType())` | accepted | accepted |

Rationale: the default build keeps path handling byte-identical to
`os`/`path/filepath` — smaller, faster, and no chance of divergence from the
standard library for a program that emulates its own OS. The tag is for
programs that need a `VFS` presenting a *foreign* OS (a Windows-looking file
system to Linux test code, and symmetrically); there the OS-parameterised
implementation is mandatory.

`Options.OSType` is honoured identically in both configurations: under the
default build a foreign value is **silently rejected** — constructors discard
the `ErrSetOSType` from `SetOSType` — so the file system keeps the host OS.
Callers who need a guarantee must check `HasFeature(FeatSetOSType)` first, or
verify `OSType()` after construction.

#### Normative rules

1. `OSType()` is the single source of truth for path handling. No path method
   may consult the host `GOOS`, and none may cache the separator or the volume
   syntax independently of `SetOSType`.
2. `FeatSetOSType` is advertised **if and only if** `SetOSType` accepts a
   foreign OS. `HasFeature(FeatSetOSType)` and `BuildFeatures()&FeatSetOSType != 0`
   agree for every emulated implementation, and real file systems
   (`FeatRealFS`) never advertise it — the host OS cannot change.
3. A successful `SetOSType` must re-derive every value cached from the previous
   OS **before returning**, and the change must be atomic with respect to
   concurrent file operations. `OSTypeMixin` owns only what it can derive
   itself — separator, `DirMode`, `FileMode`. Everything else belongs to the
   file system, which must recompute:
   - the error table (`avfs.ErrorsFor(os)`), since messages differ per OS;
   - the current user's home and temporary directories, and the current
     directory if it is no longer valid (`\Users\x` vs `/home/x`);
   - Windows volume state (`VolumeAdd`/`VolumeDelete`/`VolumeList`): the volume
     table must exist for `OsWindows` and be absent otherwise;
   - default permissions, where they differ (`dirMode` on Windows).
   A bare `OSTypeMixin.SetOSType` that leaves stale derived state behind is
   non-conforming even though the mixin itself is satisfied.
4. `SetOSType` never converts existing content. Path *names* stored in the tree
   are component-wise and remain valid, but absolute paths and symlink targets
   written under the old OS are not rewritten; callers wanting a converted view
   should build a new file system with `CloneWithOptions` (§4.1) rather than
   mutate a populated one in place.
5. `SetOSType(OsUnknown)` always resolves to `CurrentOSType()` and never fails.

An emulated identity manager is subject to the same rules: `memidm` derives its
administrator user and group names from its `OSType`
(`avfs.AdminUserName`/`AdminGroupName`), so an idm whose OS is out of sync with
its file system produces a home directory (`/Users/ContainerAdministrator` vs
`\Users\root`) that does not exist.

Build tags:

| Tag | Effect |
|-----|--------|
| `avfs_setostype` | Enables runtime OS switching: `BuildFeatures()` includes `FeatSetOSType`, `PathMixin` is OS-parameterised, and emulated file systems advertise `FeatSetOSType` |
| `avfs_race` | Runs the concurrency test suite instead of the standard one |

Both configurations are part of the contract: CI runs the whole suite on Linux,
Windows and macOS with and without `avfs_setostype`, and code that uses a
tag-gated symbol must compile in both.

### 3.7 Umask

`UMaskMixin` provides a per-file-system, lock-free umask (`atomic.Uint32`).
Package-level `SetUMask`/`UMask` manipulate the **process** umask on Unix
(via `syscall.Umask`, cached in a global guarded by a `sync.RWMutex`); on
Windows the default is `0o111`, elsewhere `0o022`.

The mask is applied at creation time: `perm & avfs.FileModeMask &^ UMask()`,
where `FileModeMask = ModePerm | ModeSticky | ModeSetuid | ModeSetgid`.

### 3.8 Identity

`IdmMgr` = `Featurer` + `OSTyper` + `Typer` + `UserMgr`, with:
`AdminGroup`, `AdminUser`, `AddGroup`, `AddUserToGroup`, `DelGroup`,
`DelUserFromGroup`, `LookupGroup`, `LookupGroupId`, `LookupUser`,
`LookupUserId`, `SetUserPrimaryGroup`.

`UserMgr`: `AddUser(userName, groupName)`, `DelUser(userName)`.
`UserReader`: `Gid`, `Uid`, `Name`, `Groups`, `GroupsId`, `IsInGroupId`,
`IsAdmin`, `PrimaryGroup`, `PrimaryGroupId`.
`GroupReader`: `Gid`, `Name`.

Conventions:

- User and group names must match `^[a-zA-Z0-9_-]+$` (`avfs.IsValidName`).
- The administrator is uid/gid `0`; the admin names are `root` on Unix and
  `ContainerAdministrator`/`Administrators` on Windows (`AdminUserName`,
  `AdminGroupName`).
- `SetIdm(nil)` installs `avfs.DefaultIdm`.

### 3.9 Error model

Errors are file-system independent and OS-shaped:

- `LinuxError uintptr` — Linux errno values
  (`ErrOpNotPermitted`, `ErrNoSuchFileOrDir`, `ErrPermDenied`, `ErrFileExists`,
  `ErrDirNotEmpty`, `ErrTooManySymlinks`, `ErrIsADirectory`,
  `ErrNotADirectory`, `ErrInvalidArgument`, `ErrBadFileDesc`, `ErrCrossDevLink`).
  Darwin reuses the Linux values.
- `WindowsError uintptr` — Windows codes (`ErrWinFileNotFound`,
  `ErrWinPathNotFound`, `ErrWinAccessDenied`, `ErrWinInvalidName`,
  `ErrWinAlreadyExists`, `ErrWinDirNotEmpty`, `ErrWinNotSupported`,
  `ErrWinNotReparsePoint`, `ErrWinPrivilegeNotHeld`, …).
- `CustomError uintptr` — VFS-specific errors that have no OS equivalent
  (`ErrNegativeOffset`, `ErrFileClosing`, `ErrPatternHasSeparator`,
  `ErrVolumeAlreadyExists`, `ErrVolumeNameInvalid`, `ErrVolumeWindows`).

All three satisfy `SysError`: `Error() string`, `No() uint`, and
`Is(target)` mapping to `fs.ErrPermission`, `fs.ErrExist`, `fs.ErrNotExist`.

`ErrorsFor(ost) *ErrorsForOS` normalises semantic errors
(`PermDenied`, `NoSuchFile`, `NoSuchDir`, `FileExists`, `DirNotEmpty`,
`IsADirectory`, `NotADirectory`, `OpNotPermitted`, `BadFileDesc`,
`InvalidArgument`, `NegativeSeek`, `FileClosing`, `TooManySymlinks`) for a given
OSType; it **panics** on an unknown `OSType`.

Identity errors are typed: `UnknownUserError`, `UnknownUserIdError`,
`UnknownGroupError`, `UnknownGroupIdError`, `AlreadyExistsUserError`,
`AlreadyExistsGroupError`, `InvalidNameError`, `UnknownError`.

Operation-name constants: `OpWinCreateFile` (`GetFileAttributesEx`),
`OpReaddirent`, `OpReaddir`.

### 3.10 Generic helper functions

Package-level generic helpers implement the derived operations once, on top of
a `VFSBase`, so that every implementation shares identical behaviour:

`Create`, `CreateTemp`, `FromUnixPath`, `Glob`, `IsExist`, `IsNotExist`,
`MkdirTemp`, `ReadDir`, `ReadFile`, `SetUserByName`, `SplitAbs`, `ToOpenMode`,
`WalkDir`, `WriteFile`.

Also provided: `MkDirs` (create a set of `DirInfo` directories), `SystemDirs`
and `UserDirs` (OS-specific directory metadata), `homeDirUser`, `tempDirUser`,
`ToSysStat`, `PathIterator[T]`, `CopyFile`, `CopyFileHash`, `HashFile`,
`Tree`, `NewRndTree`.

### 3.11 Constants

`DefaultDirPerm` `0o777`, `DefaultFilePerm` `0o666`, `DefaultName` `"Default"`,
`DefaultVolume` `"C:"`, `NotImplemented` `"not implemented"`, `FileModeMask`.

`OpenMode` bitmask used by permission checks:
`OpenLookup`, `OpenWrite`, `OpenRead`, `OpenAppend`, `OpenCreate`,
`OpenCreateExcl`, `OpenTruncate`, `OpenDir`; `ToOpenMode(flag)` converts
`os` flags to it.

`DirInfo{Path, Perm, Uid, Gid}` carries directory metadata.
`SysStater` (`Uid`, `Gid`, `Nlink`) is what `ToSysStat` returns.

`FnVFS` is a 43-value enum naming every VFS/File function, used by `failfs` to
select an operation to fail.

---

## 4. File system implementations

### 4.1 `memfs` — full-featured in-memory reference

- `New() *MemFS`, `NewWithOptions(*Options) *MemFS`
- `Options{Idm, User, Name, SystemDirs []avfs.DirInfo, OSType}`; `nil` means
  defaults (`memidm.New()`, `idm.AdminUser()`, `avfs.SystemDirs(vfs)`, host
  umask, current OSType).
- Exports `MemFS`, `MemFile`, `MemInfo` (`fs.FileInfo` + `fs.DirEntry` +
  `avfs.SysStater`), `MemIOFS` (io/fs adapter).
- Windows volume emulation: `VolumeAdd`, `VolumeDelete`, `VolumeList`.
- Internals: a node tree (`dirNode`, `fileNode`, `symlinkNode`) protected by
  per-node `sync.RWMutex`; `searchNode` resolves paths with a
  `slmLstat`/`slmStat`/`slmEval` mode and a 64-symlink loop limit
  (`ErrTooManySymlinks`).
- Features: `FeatHardlink | FeatSubFS | FeatSymlink | BuildFeatures()` plus the
  identity manager's features.
- **Runtime OS switching**: `SetOSType` is the promoted `OSTypeMixin` method,
  and under `avfs_setostype` it accepts any `OSType`. On success it must
  recompute `err` (`avfs.ErrorsFor`), the Windows volume table, and the user
  directories before returning (§3.6, rule 3) — none of these are visible in
  the mixin today, so in-place switching is currently incomplete.
- **Copying with a different identity**: `CloneWithOptions(*Options)` (planned,
  see §9) is the supported way to obtain a second view of the same tree with
  another user and/or `OSType`; it shares the node tree and the unique-id
  counter (so `SameFile` stays consistent across views) while giving the copy
  its own user, working directory and derived state.
- **Permissions are genuinely enforced** (`checkPermission` on lookup, read and
  write), and umask is applied on creation.
- Hard links share one `fileNode` (`nlink++`); `SameFile` compares the unique
  node id.
- `Type() == "MemFS"`.
- Known gap: `Sub` advertises `FeatSubFS` but currently returns `PermDenied`
  (same for `MemIOFS.Sub`).

### 4.2 `orefafs` — simplified in-memory file system

- `New() *OrefaFS`, `NewWithOptions(*Options) *OrefaFS`
- `Options{User, Name, SystemDirs, OSType}` — no `Idm` field.
- Exports `OrefaFS`, `OrefaFile`, `OrefaInfo`.
- Internals: a flat `map[string]*node` keyed by absolute path plus per-node
  children maps, guarded by a single `sync.RWMutex`.
- Features: `FeatHardlink | BuildFeatures()` only — no symlinks, no identity
  manager (`DefaultIdm`).
- Runtime OS switching follows the same rules as `memfs` (§3.6): accepted under
  `avfs_setostype`, advertised through `BuildFeatures()`, and responsible for
  refreshing its error table and user directories. It has no volume table, so
  only the error table and user directories are at stake.
- Umask is applied at creation, but **no permission checks** are performed: it
  is the permissive base for decorators such as `failfs`.
- `Symlink`, `Readlink`, `EvalSymlinks`, `Sub` fail with `PermDenied`.
- `Type() == "OrefaFS"`.

### 4.3 `osfs` — native file system

- `New() *OsFS` (no options).
- Every method is a direct `os.*`/`path/filepath.*` call; `*os.File` is
  returned as the `avfs.File`.
- Features: `FeatRealFS | FeatSymlink | FeatHardlink`; identity manager is
  `DefaultIdm`.
- `SetUser`/`SetUserByName` return `ErrPermDenied`; `Chown`/`Lchown` return
  `ErrOpNotPermitted` unless an identity manager is installed.
- `Sub` returns `PermDenied`. No `Chroot`.
- Exports per-OS `SysStat` adapters (`LinuxSysStat`, `WindowsSysStat`,
  `OtherSysStat`). `Type() == "OsFS"`.

### 4.4 `ostestfs` — native file system with OS identity

- `New() *OsTestFS` (= `NewWithOptions(&Options{Idm: osidm.New()})`),
  `NewWithOptions(*Options) *Options{Idm avfs.IdmMgr}`.
- Embeds `osfs.OsFS`; overrides `Chown`, `Lchown`, `SetUser`, `SetUserByName`,
  `User`.
- Features: `FeatRealFS | FeatSymlink | FeatHardlink | idm.Features()`.
- Implements `Chroot`: real `syscall.Chroot` on Unix (requires
  `FeatIdentityMgr`, else `ErrOpNotPermitted`), `ErrWinNotSupported` on
  Windows.
- `Type() == "OsTestFS"`.

### 4.5 `basepathfs` — base-path decorator

- `New(baseFS avfs.VFS, basePath string) *BasePathFS` (panics on error),
  `NewWithErr(baseFS, basePath) (*BasePathFS, error)` — the base path must
  exist and be a directory, otherwise `*fs.PathError` with
  `ErrNotADirectory`.
- Exports `BasePathFS`, `BasePathFile`, plus the translation helpers
  `ToBasePath`, `FromBasePath`, `FromPathError`, `FromLinkError`.
- Every path is mapped into the base directory; returned errors are rewritten
  back to the virtual namespace.
- Symlinks are disabled by construction: `FeatSymlink` is stripped from the
  base feature set and `Symlink`/`Readlink`/`EvalSymlinks` fail with
  `PermDenied` (`ErrWinAccessDenied`/`ErrWinNotReparsePoint` on Windows).
- `Sub` is forwarded to the base file system.
- `Type() == "BasePathFS"`.

### 4.6 `rofs` — read-only decorator

- `New(baseFS avfs.VFS) *RoFS`; exports `RoFS`, `RoFile`.
- Features: `base.Features() &^ FeatIdentityMgr | FeatReadOnly`.
- All mutating operations return `PermDenied` (`*fs.PathError`, or
  `*os.LinkError` for `Link`/`Rename`/`Symlink`), including `File.Write`,
  `File.WriteAt`, `File.Truncate`, `File.Sync`, `File.Chmod`, `File.Chown`,
  `File.Chdir`, and `SetIdm`/`SetUser`.
- `OpenFile` accepts only `os.O_RDONLY`.
- Read operations are forwarded unchanged.
- `Type() == "RoFS"`.

### 4.7 `failfs` — fault-injection decorator

- `New(baseFS avfs.VFS) *FailFS`;
  `SetFailFunc(FailFunc) error`.
- `type FailFunc func(vfs avfs.VFSBase, fn avfs.FnVFS, failParam *FailParam) error`
- `FailParam{ATime, MTime, Op, Path, NewPath, Flag, Uid, Gid, Size, Perm}`.
- Provided functions: `OkFunc` (never fails) and `ReadOnlyFunc` (fails all
  writes with the correct `*fs.PathError`/`*os.LinkError` shapes for the
  emulated OS).
- Every VFS and File operation first calls `fail(...)`; a non-nil error
  short-circuits, otherwise the call is forwarded to the base file system.
  Utility functions (`Clean`, `Join`, `Match`, …) are never failed.
- `Type() == "FailFS"`.

---

## 5. Identity manager implementations

### 5.1 `DefaultIdm` / `DummyIdm` (in package `avfs`)

`avfs.DefaultIdm` is the default identity manager, an instance of the
`DummyIdm` type. It reports no features, uses `DefaultName` with
`math.MaxInt` ids, and every lookup or mutation returns `ErrPermDenied`.
Constructors: `NewDummyIdm`, `NewUser`, `NewGroup`.
`Type() == "DummyIdm"`.

### 5.2 `memidm` — in-memory identity manager

- `New() *MemIdm`, `NewWithOptions(*Options) *Options{OSType avfs.OSType}`.
- Exports `MemIdm`, `MemUser`, `MemGroup`.
- Administrator is uid/gid `0`; new users and groups start at `1000` and
  increment.
- Features: exactly `FeatIdentityMgr`.
- Names are validated with `avfs.IsValidName` (`InvalidNameError`);
  duplicates give `AlreadyExistsUserError`/`AlreadyExistsGroupError`;
  missing entries give `UnknownUserError`/`UnknownGroupError`/
  `UnknownUserIdError`/`UnknownGroupIdError`.
- `MemUser.IsAdmin()` is `uid == 0 || IsInGroupId(0)`.
- **Not safe for concurrent use**; its tests are excluded under `avfs_race`.

### 5.3 `osidm` — OS-backed identity manager (testing only)

- `New() *OsIdm`; exports `OsIdm`, `OsUser`, `OsGroup`.
- Linux: `groupadd`, `useradd -M -g`, `usermod -aG`, `userdel`, `groupdel`,
  `getent`. macOS: `dscl`/`dseditgroup` and `sysadminctl addUser`. Other
  platforms: every mutation returns `ErrPermDenied`.
- Features: `FeatIdentityMgr`, plus `FeatReadOnlyIdm` when the process is not
  root. On Windows the feature set is `0` and ids are `math.MaxInt`.
- Command stderr is parsed into typed errors, otherwise `UnknownError`.
- `Type() == "OsIdm"`; `OSType()` returns `CurrentOSType()`.

---

## 6. Conformance test suite (`test` package)

### 6.1 Structure

A `test.Suite` holds **two** file system handles:

- `vfsSetup` — a writable/privileged file system used to create fixtures,
- `vfsTest` — the system under test.

This separation is what allows read-only, fault-injecting and
identity-switching implementations to be validated while fixtures are still
created normally.

Constructors:

- `NewSuiteFS(tb, vfsSetup, vfsTest avfs.VFSBase) *Suite`
- `NewSuiteIdm(tb, idm avfs.IdmMgr) *Suite` — wraps a bare identity
  manager in an `ostestfs` file system so it can be exercised through the full
  VFS suite.

Suite invariants:

1. If `vfsTest.OSType() != avfs.CurrentOSType()` the suite is **skipped** —
   OS-specific error semantics can only be validated natively.
2. `vfsTest.User()` must be non-nil.
3. `canTestPerm` is true only when the OS is not Windows, the initial user is
   an administrator, `FeatIdentityMgr` is set and `FeatReadOnlyIdm` is not.
4. On Linux the umask is forced to `0o22` so permission baselines match.
5. Fixture groups and users are created when the identity manager is writable.

### 6.2 Drivers

- `TestVFSAll(t)` = `TestVFS` + `TestFile` + `TestUtils`.
- `TestVFS(t)` runs the suite twice: once as the unprivileged user
  (`UsrTest`), and once as the administrator (root) for `Chmod`, `Chown`,
  `Lchown`, `Chroot`, `MkSystemDirs`, `SetUserByName`, `Volume` and
  `WriteOnReadOnlyFS`.
- `TestFile(t)` — per-file operation tests (read, write, seek, truncate, sync,
  `Fd`, `ReadDir`, `Readdirnames`, `ReaderFrom`/`WriterTo`, …).
- `TestUtils(t)` — path and utility helpers (`Join`, `Rel`, `Match`,
  `FromUnixPath`, `Glob`, `CopyFile`, `HashFile`, `RndTree`, …).
- `TestIdmAll(t)` — identity manager tests (admin, groups, lookups, users,
  user groups).
- `TestRace(t)` — concurrency validation, see §6.6.
- `BenchAll(b)` — create/read/write/mkdir/open/remove benchmarks.

`RunTests(t, userName, testFuncs...)` is the core loop: for each test function
it creates a fresh directory named after the function, changes into it, runs
the test as a subtest, then removes the directory as the initial user.
`RunBenchmarks` follows the same protocol.

Fixture builders include `createDir`, `createFile`, `emptyFile`, `existingDir`,
`existingFile`, `nonExistingFile`, `openedEmptyFile`, `openedNonExistingFile`,
`closedFile`, `randomDir` (via `avfs.NewRndTree` with 3 dirs, 11 files,
4 symlinks), `setUser`, `setInitUser`.

### 6.3 Fixtures identities

Groups: `grpTest`, `grpOther`, `grpEmpty`.
Users: `UsrTest` (primary group `grpTest`), `UsrGrp` (member of `grpTest`),
`UsrOth` (member of `grpOther`). Names get a random suffix to avoid collisions.

### 6.4 Expected-error framework

`test.AssertPathError(err)` / `test.AssertLinkError(err)` return a fluent
assertion builder:

```go
test.AssertPathError(err).Err(avfs.ErrPermDenied).Op("open").Path("/x").Test()
```

- Chains: `Err`, `NoError`, `Op`, `Path`, `Old`, `New`, `Test`.
- Qualifiers: `GoVersion(min, max)` and `OSType(ost)`. `Test()` is a no-op
  unless the running Go version and OS match, which is how a single assertion
  expresses OS-specific expectations.
- Shortcuts: `ErrPermDenied()` (`ErrWinAccessDenied` on Windows) and
  `ErrFileNotFound()` (`ErrWinFileNotFound` on Windows).
- Mismatches on version-bounded expectations are logged as warnings rather than
  failures.
- Also available: `test.AssertPanic`, `test.RequireNoError`, `test.AssertNoError`.

### 6.5 Permission matrix tests

`test.NewPermTests(t, testDir, funcName)` builds, for every fixture user and
every mode `0..0o777`, a directory `<permDir>/<user>/<mode>`, then invokes the
operation under test. Expected results are stored as JSON golden files in
`test/testdata/perm-<funcName>.golden`, keyed by `"<user>/<perm>"`, each value
recording the error type (`LinkError`, `PathError`, `StringError`), operation,
paths and underlying error, with absolute paths normalised away.

`PermOptions{IgnoreOp, IgnorePath, CreateFiles}` relax comparisons where the
recorded baseline legitimately differs (e.g. Windows or a real OS).

Golden files exist for: `chdir`, `chmod`, `chown`, `chtimes`, `create`,
`lchown`, `link`, `mkdir`, `mkdirall`, `openfile-dir`, `openfile-read`,
`openfile-write`, `remove`, `removeall`, `rename`, `symlink`.

### 6.6 Concurrency tests

`TestRace(t)` skips unless the emulated OS matches the host, then runs:
`RaceCreate`, `RaceCreateTemp`, `RaceFileClose`, `RaceMkdir`, `RaceMkdirAll`,
`RaceMkdirTemp`, `RaceOpen`, `RaceOpenFile`, `RaceOpenFileExcl`, `RaceRemove`,
`RaceRemoveAll`, `RaceMkdirRemoveAll`.

Each spawns 100 goroutines per function behind a `sync.RWMutex` start barrier,
counts outcomes with `atomic.Uint32`, and asserts an exact
success/failure split (`RaceResult`: `RaceNoneOk`, `RaceOneOk`, `RaceAllOk`,
`RaceUndefined`). Intended to run with `-race`.

### 6.7 Benchmarks

`BenchAll` covers create, file read, file write, mkdir, open file and remove
with 32 KiB buffers and up to 32 MiB files; `benchOpenFlags()` adds
`O_DIRECT` on a real Linux file system to bypass the page cache.

### 6.8 Cross-compilation check

`test/testbuild` is a `package main` that references every `VFS`,
`IdmMgr`, `UserReader` and `GroupReader` method for every implementation,
so `gox` cross-compilation proves all OS-specific syscall paths compile.

---

## 7. Behavioural requirements

Any conforming implementation must satisfy:

1. **Path semantics** — all path manipulation follows the emulated
   `OSType`, not the host: separators, volume names, absolute-path rules,
   `Clean`, `Join`, `Rel`, `Match`, `FromSlash`/`ToSlash`.
2. **Error shape** — path operations return `*fs.PathError`, link operations
   (`Link`, `Rename`, `Symlink`) return `*os.LinkError`; the wrapped error is
   the value returned by `avfs.ErrorsFor(ost)` for the semantic condition.
3. **Error classification** — every error satisfies `errors.Is` against
   `fs.ErrPermission`, `fs.ErrExist` and `fs.ErrNotExist` consistently with the
   emulated OS.
4. **Umask** — creation modes are masked by `UMask()` restricted to
   `FileModeMask`.
5. **Permissions** — when `FeatIdentityMgr` is set and the identity manager is
   writable, `rwx` decisions must follow the standard Unix model for both
   owner, group and others.
6. **Feature honesty** — an operation must succeed if and only if its feature
   flag is set. Setting `FeatReadOnly` obliges the implementation to reject all
   mutations.
7. **Read-only semantics** — under `FeatReadOnly`, every mutating `VFS` and
   `File` method fails with `PermDenied`; `OpenFile` accepts only `O_RDONLY`.
8. **Symlink semantics** — with `FeatSymlink`, `Lstat` does not follow links,
   `Stat`/`Open` do, `EvalSymlinks` resolves them, `Readlink` returns the raw
   (possibly relative) destination, and resolution is bounded (64 hops).
9. **Hardlink semantics** — with `FeatHardlink`, `Link` shares the underlying
   node and increments the link count; `SameFile` reports the shared identity;
   linking a directory or a symlink is refused.
10. **Chroot semantics** — when `ChRooter` is implemented, `Chroot` requires
    administrator rights and confines subsequent operations.
11. **Concurrency** — concurrent use of the same file system must be safe for
    implementations advertising concurrency (all in-memory implementations use
    mutexes; `memidm` is the documented exception).
12. **Identity isolation** — each `VFS` instance carries its own current user,
    working directory, home directory and temporary directory; `SetUser`
    recomputes them.

---

## 8. Usage

```go
package main

import (
	"os"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/osfs"
)

func main() {
	var vfs avfs.VFS

	switch os.Getenv("ENV") {
	case "PROD":
		vfs = osfs.New()
	default:
		vfs = memfs.New()
	}

	rootDir, _ := vfs.MkdirTemp("", "avfs")
	defer vfs.RemoveAll(rootDir)

	path := vfs.Join(rootDir, "aFile.txt")
	_ = vfs.WriteFile(path, []byte("randomContent"), avfs.DefaultFilePerm)

	slPath := vfs.Join(rootDir, "aFileSymlink.txt")
	_ = vfs.Symlink(path, slPath)

	content, _ := vfs.ReadFile(slPath)
	_ = content
}
```

Porting existing code is a matter of replacing the references to `os` and
`path/filepath` with the file system variable, and initialising it with the
desired implementation and options.

---

## 9. Known gaps

- `memfs` advertises `FeatSubFS` but `Sub` (and `MemIOFS.Sub`) returns
  `PermDenied`.
- `osfs.NewWithNoIdm()`, referenced by the readme and by `ostestfs`
  documentation, does not exist; use `osfs.New()`.
- `memidm` is not safe for concurrent use.
- Windows support is described in the readme as "almost ready".
