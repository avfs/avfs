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
| `mage` | Build tooling (minigox cross-compilation, lint, release) |

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
| `UserDirMixin` | `VFSUserDir` | current user, home and temp dirs (immutable), cwd (`atomic.Pointer[string]`) | `IdmMixin`, `PathMixin` |

`UserDirMixin.Init(ost, idm, user)` initialises the whole chain — OS type,
identity manager, user, home and temporary directories, current directory — and
must be called once, during construction. The current directory is the home
directory of the user, so that directory must exist. Every other setter of the
chain (`SetFeatures`, `SetUMask`) is a constructor-level operation too; there is
no way to change the identity of a live file system (see §3.3).

`memfs` and `orefafs` hold their mixin in a **named `userDir` field** rather
than embedding it, so that the state private to a file system is explicit and a
clone is obviously a new value rather than a copy of a struct holding an atomic
pointer. The `VFSPath`, `Featurer`, `UMasker`, `OSTyper`, `IdmProvider` and
`VFSUserDir` methods are then written as one-line forwarders
(`memfs_mixins.go`, `orefafs_mixins.go`) and must be kept in sync with those
interfaces. Decorators (`RoFS`, `FailFS`, `BasePathFS`) embed only
`FeaturesMixin`, since they delegate the rest to a wrapped base file system.

### 3.1 `VFS` and `VFSBase`

`VFSBase` is the union of the small capability interfaces:

| Embedded interface | Methods |
|--------------------|---------|
| `Featurer` | `Features() Features`, `HasFeature(Features) bool` |
| `IdmProvider` | `Idm() IdmMgr`, `InitIdm(IdmMgr) error` (construction only) |
| `Namer` | `Name() string` |
| `OSTyper` | `OSType() OSType` |
| `Typer` | `Type() string` |
| `UMasker` | `SetUMask(fs.FileMode) error`, `UMask() fs.FileMode` |
| `VFSPath` | path manipulation utilities (see §3.4) |
| `VFSUserDir` | `Abs`, `Getwd`, `TempDir`, `User` |

`VFSBase` additionally declares the operating-system operations:
`Base`, `Chdir`, `Chmod`, `Chown`, `Chtimes`, `Create`, `CreateTemp`,
`EvalSymlinks`, `Glob`, `UserHomeDir`, `Lchown`, `Link`, `Lstat`,
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

- `Cloner` — `CloneWithUser(user UserReader, ost OSType) (VFS, error)` and
  `CloneWithUserName(userName string, ost OSType) (VFS, error)`.
- `ChRooter` — `Chroot(path string) error`, changes the file system root
  (requires privileges; `ErrPermDenied` otherwise).

#### Identity is immutable, cloning is how it varies

A file system is built for one identity: one current user, one identity
manager, one emulated OS type, and the home and temporary directories those
imply. There is no `SetUser`, `SetUserByName` or `SetOSType` to change any of
them afterwards. The reasons are:

- **No TOCTOU on permissions.** Every operation resolves the user once. If the
  user could be swapped mid-flight, a check could pass as one user and a write
  land as another.
- **No data race.** The user, home and temp directories are plain fields read
  by every operation; making them immutable removes the need to lock them. The
  only mutable part of an identity is the current directory, which is an
  `atomic.Pointer[string]`, because `Chdir` is inherently a mutation.
- **Reproducibility.** A file system built for a Windows user and a Linux user
  are two values, not one value observed at two moments.

Cloning is therefore the only way to obtain a file system acting as another
user or emulating another OS. The rules:

1. A clone **shares the content** of the file system it is copied from (its
   nodes, its volumes, its name), so a file created through one view is visible
   from all of them, and file identity (`SameFile`) stays consistent across
   views: the unique-id counter is part of the shared state.
2. A clone has **its own identity**: its own view of the user, home and
   temporary directories, error table and umask. It starts in the home
   directory of the user it acts as, never in the current directory of the file
   system it is cloned from.
3. `memfs` and `orefafs` go one step further and **memoise the identity views**
   of their storage, keyed by user (name *and* uid) and OS type. Cloning twice
   for the same user and OS type returns two file systems sharing one identity
   view, hence one current directory: `Chdir` on one is visible in the other.
   The uid is part of the key because a deleted user may be created again with
   the same name and a different uid, and must not inherit its predecessor's
   directories.
4. A clone **creates no directory but the home directory of the user** it acts
   as, and only if it does not exist yet: the clone starts in it, and a
   directory that does not exist is not a working directory. Nothing is created
   for the temporary directory, and nothing at all is created when the clone
   emulates a foreign OS (rule 6). Since a view may have no privilege to
   create its own home directory, nor the right to own it, the creation is done
   by the administrator of the identity manager.
5. Cloning with `OsUnknown` keeps the OS type of the source; with a `nil`
   user it uses the administrator of the identity manager. Cloning with a
   foreign OS type in a build without `avfs_setostype` returns
   `ErrSetOSType`.
6. The **content is not converted** to the new OS type. A view emulating
   another OS resolves paths with the rules, and reports the errors, of that
   OS, but the directories already created keep the names and the layout of the
   OS the content was built for. A file system scoped to a path (`Sub`, a
   decorator) may consequently fail to find its own base path in the new OS —
   with a `*fs.PathError`, not with a refusal.
7. A decorator implementing `Cloner` clones its base file system and re-wraps
   the copy; if the base is not clonable it returns `ErrNotSupported`.
8. A **real** file system (`FeatRealFS`) has no view to give: its user is the
   user of the process running it, so it is never clonable and never has more
   than one identity. `ostestfs` exposes `SetUser`/`SetUserByName` for the sole
   purpose of changing the credentials of the process (a real `setresuid`/`setresgid`
   through `osidm`); they are not file system operations, and that is why the
   permission tests of the conformance suite — the reference the golden files
   are recorded from — must run as root.

The conformance suite runs its tests as a given user in one of two ways, chosen
by what the file system under test supports:

- a **clonable** file system is asked for a view acting as that user, for the
  setup file system as well, so that fixtures are owned by the user that uses
  them;
- a **real** file system changes the credentials of the process
  (`userSwitcher`, see rule 8).

A file system that supports neither is a test setup error: permission tests
would silently not run.

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

The `avfs_setostype` implementation is a port of the standard library
`path/filepath` and `internal/filepathlite` code of **Go 1.27**, where the
compile-time `Separator`, `IsPathSeparator` and `volumeNameLen` decisions are
replaced by the emulated `OSType`. It therefore recognizes the Windows device
prefixes (`\\.\`, `\\?\`, `\??\`) and rejects a volume name containing a `..`
path element, like the standard library does. It must be refreshed when a
newer Go version fixes the ported functions.

### 3.5 Feature flags

`Features` is a `uint64` bitmask:

| Flag | Meaning |
|------|---------|
| `FeatHardlink` | `Link` is supported |
| `FeatIdentityMgr` | An identity manager exists; multiple users supported |
| `FeatSetOSType` | The emulated OS may differ from the host (build tag `avfs_setostype`, see §3.6) |
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
was built with `avfs_setostype`, and then only if `InitOSType` and
`CloneWithUser` honour the rules of §3.6.

### 3.6 Operating system emulation

`OSType` is one of `OsUnknown`, `OsLinux`, `OsWindows`, `OsDarwin`.
`OSTypeMixin` (embeddable) provides `OSType`, `InitOSType`, `PathSeparator`,
`DirMode`, `FileMode`. `InitOSType` is a construction-time operation: the OS
type of a file system never changes, and another one is obtained by cloning
(§3.3). `CurrentOSType()` reports the host OS.

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
| `InitOSType(foreignOS)` | `ErrSetOSType` | accepted |
| `InitOSType(OsUnknown)` / `InitOSType(CurrentOSType())` | accepted | accepted |
| `CloneWithUser(u, foreignOS)` | `ErrSetOSType` | a view of the content for that OS |

Rationale: the default build keeps path handling byte-identical to
`os`/`path/filepath` — smaller, faster, and no chance of divergence from the
standard library for a program that emulates its own OS. The tag is for
programs that need a `VFS` presenting a *foreign* OS (a Windows-looking file
system to Linux test code, and symmetrically); there the OS-parameterised
implementation is mandatory.

`Options.OSType` is honoured identically in both configurations: under the
default build a foreign value is **silently rejected** — constructors discard
the `ErrSetOSType` from `InitOSType` — so the file system keeps the host OS.
Callers who need a guarantee must check `HasFeature(FeatSetOSType)` first, or
verify `OSType()` after construction.

#### Normative rules

1. `OSType()` is the single source of truth for path handling. No path method
   may consult the host `GOOS`, and none may cache the separator or the volume
   syntax independently of `InitOSType`.
2. `FeatSetOSType` is advertised **if and only if** a foreign OS type is
   honoured, whether at construction or through `CloneWithUser`.
   `HasFeature(FeatSetOSType)` and `BuildFeatures()&FeatSetOSType != 0` agree for
   every emulated implementation, and real file systems (`FeatRealFS`) never
   advertise it — the host OS cannot change.
3. Every value derived from the OS type is derived **once, at construction**, so
   that it can never be stale: `OSTypeMixin.InitOSType` computes the separator
   and the default modes, and the file system computes
   - the error table (`avfs.ErrorsFor(os)`), since messages differ per OS;
   - the home and temporary directories of the current user, the home
     directory being also the initial current directory (`\Users\x` vs
     `/home/x`);
   - Windows volume state (`VolumeAdd`/`VolumeDelete`/`VolumeList`): the volume
     table exists for `OsWindows` and is absent otherwise.

   An emulated identity manager derives its administrator user and group names
   from its own OS type (`avfs.AdminUserName`/`AdminGroupName`), so an idm whose
   OS is out of sync with its file system yields a home directory
   (`/Users/ContainerAdministrator` vs `\Users\root`) that does not exist.
   `memidm` therefore includes `BuildFeatures()` in its features, so that
   `FeatSetOSType` tells whether its OS type was honoured.
4. Changing the OS type of a populated file system is not a mutation: it is a
   clone (§3.3, rule 6), which never converts the content.
5. `InitOSType(OsUnknown)` always resolves to `CurrentOSType()` and never fails.

An emulated identity manager is subject to the same rules: `memidm` derives its
administrator user and group names from its `OSType`
(`avfs.AdminUserName`/`AdminGroupName`), so an idm whose OS is out of sync with
its file system produces a home directory (`/Users/ContainerAdministrator` vs
`\Users\root`) that does not exist.

Build tags:

| Tag | Effect |
|-----|--------|
| `avfs_setostype` | Enables emulating a foreign OS: `BuildFeatures()` includes `FeatSetOSType`, `PathMixin` is OS-parameterised, and emulated file systems and identity managers advertise `FeatSetOSType` |
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
- `InitIdm(nil)` installs `avfs.DefaultIdm`. `InitIdm` is a construction-time
  operation: the users a file system resolves never change under it.

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
`MkdirTemp`, `ReadDir`, `ReadFile`, `SplitAbs`, `ToOpenMode`,
`WalkDir`, `WriteFile`.

Also provided: `MkDirs` (create a set of `DirInfo` directories), `SystemDirs`,
`UserDirs` and `HomeDirInfo` (OS-specific directory metadata), `homeDirUser`, `tempDirUser`,
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

- `New() (*MemFS, error)`, `NewWithOptions(*Options) (*MemFS, error)`
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
- `Storage` holds everything shared with its clones: the node tree, the
  unique-id counter (`lastId`), the Windows volume table (guarded by
  `volMu`), the file system name, and the memoised identity views
  (`userDirs`, guarded by `udMu`, keyed by user name, uid and OS type — see
  §3.3, rule 3). `MemFS` itself holds only its error table and its `userDir`
  view.
- Implements `Cloner`: `CloneWithUser`, `CloneWithUserName`. The constructor
  always builds the tree as the administrator (the system and user directories
  need privileges to be created and chowned) and then hands it over with a
  clone when `Options.User` is not the administrator.
- The identity views are created with the features and the umask of the file
  system they are cloned from: those describe the content and the creation
  policy, not the identity. The home directory of the cloned user is created by
  the administrator, through a view of the storage emulating the same OS type.
- **Permissions are genuinely enforced** (`checkPermission` on lookup, read and
  write), and umask is applied on creation.
- Hard links share one `fileNode` (`nlink++`); `SameFile` compares the unique
  node id, which is why the counter is part of the shared `Storage`.
- `Sub(dir)` returns a `basepathfs` scoped to `dir`: the subtree shares the
  content, and the identity of the file system. `MemIOFS.Sub` returns its
  `io/fs` projection.
- `Type() == "MemFS"`.

### 4.2 `orefafs` — simplified in-memory file system

- `New() (*OrefaFS, error)`,
  `NewWithOptions(*Options) (*OrefaFS, error)`
- `Options{User, Name, SystemDirs, OSType}` — no `Idm` field.
- Exports `OrefaFS`, `OrefaFile`, `OrefaInfo`.
- Internals: a flat `map[string]*node` keyed by absolute path plus per-node
  children maps, guarded by a single `sync.RWMutex`.
- `Storage` holds what its clones share: the map of nodes, the unique-id
  counter and that lock. `OrefaFS` holds its error table and its `userDir`
  view. Unlike `memfs`, it does not memoise identity views: each clone gets a
  fresh one, so clones of the same user have independent current directories.
- Features: `FeatHardlink | BuildFeatures()` only — no symlinks, no identity
  manager (`DefaultIdm`).
- Implements `Cloner`: a clone starts in the home directory of the user it
  acts as, which it creates in the shared content if it does not exist yet.
- Emulating a foreign OS follows the same rules as `memfs` (§3.6). It has no
  volume table, so only the error table and the user directories are derived
  from the OS type.
- Umask is applied at creation, but **no permission checks** are performed: it
  is the permissive base for decorators such as `failfs`.
- `Symlink`, `Readlink`, `EvalSymlinks`, `Sub` fail with `PermDenied`.
- `Type() == "OrefaFS"`.

### 4.3 `osfs` — native file system

- `New() (*OsFS, error)` (no options).
- Every method is a direct `os.*`/`path/filepath.*` call; `*os.File` is
  returned as the `avfs.File`.
- Features: `FeatRealFS | FeatSymlink | FeatHardlink`; identity manager is
  `DefaultIdm`.
- The identity of a real file system is the identity of the process: there is
  no way to change the user, so a real file system is never clonable.
  `Chown`/`Lchown` return `ErrOpNotPermitted` unless an identity manager is
  installed.
- `Sub` returns `PermDenied`. No `Chroot`.
- Exports per-OS `SysStat` adapters (`LinuxSysStat`, `WindowsSysStat`,
  `OtherSysStat`). `Type() == "OsFS"`.

### 4.4 `ostestfs` — native file system with OS identity

- `New() (*OsTestFS, error)`
  (= `NewWithOptions(&Options{Idm: osidm.New()})`),
  `NewWithOptions(*Options) (*OsTestFS, error)`;
  `Options{Idm avfs.IdmMgr}`.
- Embeds `osfs.OsFS`; overrides `Chown`, `Lchown` and `User`.
- Not clonable: its user is the user of the process. `SetUser`/`SetUserByName`
  change the credentials of the process through `osidm` and exist only to run
  the permission tests (§3.3, rule 8).
- Features: `FeatRealFS | FeatSymlink | FeatHardlink | idm.Features()`.
- Implements `Chroot`: real `syscall.Chroot` on Unix (requires
  `FeatIdentityMgr`, else `ErrOpNotPermitted`), `ErrWinNotSupported` on
  Windows.
- `Type() == "OsTestFS"`.

### 4.5 `basepathfs` — base-path decorator

- `New(baseFS avfs.VFS, basePath string) (*BasePathFS, error)` — the base path must
  exist and be a directory, otherwise `*fs.PathError` with
  `ErrNotADirectory`.
- Exports `BasePathFS`, `BasePathFile`, plus the translation helpers
  `ToBasePath`, `FromBasePath`, `FromPathError`, `FromLinkError`.
- Every path is mapped into the base directory; returned errors are rewritten
  back to the virtual namespace. `Getwd` returns the base path itself when the
  base file system stands outside the base path, which a clone does: it starts
  in the home directory of its user (§3.3, rule 2).
- Symlinks are disabled by construction: `FeatSymlink` is stripped from the
  base feature set and `Symlink`/`Readlink`/`EvalSymlinks` fail with
  `PermDenied` (`ErrWinAccessDenied`/`ErrWinNotReparsePoint` on Windows).
- `Sub` is forwarded to the base file system.
- Implements `Cloner`: the base file system is cloned and the copy is scoped to
  the same base path. If the base is not clonable, it returns `ErrNotSupported`.
- `Type() == "BasePathFS"`.

### 4.6 `rofs` — read-only decorator

- `New(baseFS avfs.VFS) (*RoFS, error)`; exports `RoFS`, `RoFile`.
- Features: `base.Features() &^ FeatIdentityMgr | FeatReadOnly`.
- All mutating operations return `PermDenied` (`*fs.PathError`, or
  `*os.LinkError` for `Link`/`Rename`/`Symlink`), including `File.Write`,
  `File.WriteAt`, `File.Truncate`, `File.Sync`, `File.Chmod`, `File.Chown`,
  `File.Chdir`, and `InitIdm`.
- `OpenFile` accepts only `os.O_RDONLY`.
- Read operations are forwarded unchanged.
- `Type() == "RoFS"`.

### 4.7 `failfs` — fault-injection decorator

- `New(baseFS avfs.VFS) (*FailFS, error)`;
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
  `Lchown`, `Chroot`, `MkSystemDirs`, `Volume` and
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
`RaceCloneWithUserName`, `RaceCreate`, `RaceCreateTemp`, `RaceFileClose`,
`RaceMkdir`, `RaceMkdirAll`, `RaceMkdirTemp`, `RaceOpen`, `RaceOpenFile`,
`RaceOpenFileExcl`, `RaceRemove`, `RaceRemoveAll`, `RaceMkdirRemoveAll`.

`RaceCloneWithUserName` covers the concurrency of the identity views: goroutines
cloning the *same* user (one shared view, so one current directory) and
goroutines cloning *different* users (views created concurrently, with the
identity manager populated beforehand, as `memidm` is not required to be safe
for concurrent use).

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
so `minigox` cross-compilation proves all OS-specific syscall paths compile.

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
12. **Identity immutability** — the user, home directory, temporary directory,
    identity manager and emulated OS type of a `VFS` are fixed at construction
    and never change: there is no setter, so no operation can observe the
    identity of a file system changing under it. Another identity is obtained by
    cloning (§3.3). The current directory is the one exception and is safe to
    change concurrently.
13. **Clone transparency** — clones of a file system share its content, so a
    change made through one is visible from all of them, and file identity
    (`SameFile`) is consistent across clones. A clone changes nothing else: it
    rewrites no path, and creates nothing but the home directory of its user if
    it is missing.

---

## 8. Usage

```go
package main

import (
	"log"
	"os"

	"github.com/avfs/avfs"
	"github.com/avfs/avfs/vfs/memfs"
	"github.com/avfs/avfs/vfs/osfs"
)

func main() {
	var (
		vfs avfs.VFS
		err error
	)

	switch os.Getenv("ENV") {
	case "PROD":
		vfs, err = osfs.New()
	default:
		vfs, err = memfs.New()
	}
	if err != nil {
		log.Fatal(err)
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

- Cloning with a foreign OS type does not convert the content: a Windows view of
  a tree built for Linux resolves paths with Windows rules over a Unix-shaped
  tree, and creates no directory for it (§3.3, rules 4 and 6). Building a file
  system with `Options.OSType` is the supported way to get a consistent one.
- `memfs` clones memoise their identity views (one current directory per user
  and OS type); `orefafs` clones do not, so their current directories are
  independent. The two are not consistent with each other yet.
- `CloneWithUserName` cannot tell an unknown user from a user the identity
  manager refuses to disclose: both surface as the error of `LookupUser`
  (`UnknownUserError` or `ErrPermDenied`).
- `osfs.NewWithNoIdm()`, referenced by the readme and by `ostestfs`
  documentation, does not exist; use `osfs.New()`.
- `memidm` is not safe for concurrent use.
- Windows support is described in the readme as "almost ready".
