
# Plugin Interface 

To write a plugin for an ecosystem, implement the plugin interface:

```golang
type Plugin interface {
	SetRootModule(path string) error
	GetVersion() (string, error)
	GetMetadata() Metadata
	GetRootModule(path string) (*meta.Package, error)
	ListUsedModules(path string) ([]meta.Package, error)
	ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error)
	IsValid(path string) bool
	HasModulesInstalled(path string) error
}
```

## Plugin Interface Functions Reference

Here is an overview of what each of the methods do

### SetRootModule(path string) 

Sets the path to the topmost directory where the code lives, ie in golang this is where go.mod lives or Cargo.toml in rust/cargo.

### GetVersion()

Returns the version of the underlying ecosystem tooling

### GetMetadata()

Returns a `plugin.Metadata` struct with the underlying ecosystem data
abstracted to the common format.

### GetRootModule(path string) (*meta.Package, error)

???

### ListUsedModules(path string) ([]meta.Package, error)

Returns a list of `meta.Package`s representing the first level of dependencies (???)

### ListModulesWithDeps(path string, globalSettingFile string) ([]meta.Package, error)

Returns a list of `meta.Package`s with the full dependency tree

### IsValid(path string) bool

Returns a bool indicating if the plugin has everything it needs to run

### HasModulesInstalled(path string) error

Returns an error if the underlying filesystem is not yet able to compute
the dependency graph.

