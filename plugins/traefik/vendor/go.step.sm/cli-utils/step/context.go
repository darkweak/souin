package step

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"go.step.sm/cli-utils/errs"
	"go.step.sm/cli-utils/ui"
)

// IgnoreEnvVar is a value added to a flag EnvVar to avoid the use of
// environment variables or configuration files.
const IgnoreEnvVar = "STEP_IGNORE_ENV_VAR"

// Context represents a Step Path configuration context. A context is the
// combination of a profile and an authority.
type Context struct {
	Name      string `json:"-"`
	Profile   string `json:"profile"`
	Authority string `json:"authority"`
	config    map[string]interface{}
}

// Validate validates a context and returns an error if invalid.
func (c *Context) Validate() error {
	suffix := "; check your $STEPPATH/contexts.json file"
	if c == nil {
		return errors.Errorf("context cannot be nil%s", suffix)
	}
	if c.Authority == "" {
		return errors.Errorf("context cannot have an empty authority value%s", suffix)
	}
	if c.Profile == "" {
		return errors.Errorf("context cannot have an empty profile value%s", suffix)
	}
	return nil
}

// Path return the base path relative to the context.
func (c *Context) Path() string {
	return filepath.Join(BasePath(), "authorities", c.Authority)
}

// ProfilePath return the profile base path relative to the context.
func (c *Context) ProfilePath() string {
	return filepath.Join(BasePath(), "profiles", c.Profile)
}

// DefaultsFile returns the location of the defaults file for the context.
func (c *Context) DefaultsFile() string {
	return filepath.Join(c.Path(), "config", "defaults.json")
}

// ProfileDefaultsFile returns the location of the defaults file at the base
// of the profile path.
func (c *Context) ProfileDefaultsFile() string {
	return filepath.Join(c.ProfilePath(), "config", "defaults.json")
}

// Load loads the configuration for the given context.
func (c *Context) Load() error {
	c.config = map[string]interface{}{}
	for _, f := range []string{c.DefaultsFile(), c.ProfileDefaultsFile()} {
		b, err := os.ReadFile(f)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return errs.FileError(err, f)
		}

		values := make(map[string]interface{})
		if err := json.Unmarshal(b, &values); err != nil {
			return errors.Wrapf(err, "error parsing %s", f)
		}

		for k, v := range values {
			c.config[k] = v
		}
	}

	attributesBannedFromConfig := []string{
		"context",
		"profile",
		"authority",
	}
	for _, attr := range attributesBannedFromConfig {
		if _, ok := c.config[attr]; ok {
			ui.Printf("cannot set '%s' attribute in config files\n", attr)
			delete(c.config, attr)
		}
	}

	return nil
}

// ContextMap represents the map of available Contexts that is stored
// at the base of the Step Path.
type ContextMap map[string]*Context

type storedCurrent struct {
	Context string `json:"context"`
}

// CtxState is the type that manages context state for the cli.
type CtxState struct {
	current  *Context
	contexts ContextMap
	config   map[string]interface{}
}

var ctxState = &CtxState{}

// Init initializes the context map and current context state.
func (cs *CtxState) Init() error {
	if err := cs.initMap(); err != nil {
		return err
	}
	if err := cs.initCurrent(); err != nil {
		return err
	}
	return cs.load()
}

func (cs *CtxState) initMap() error {
	contextsFile := ContextsFile()
	b, err := os.ReadFile(contextsFile)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errs.FileError(err, contextsFile)
	}
	cs.contexts = ContextMap{}
	if err := json.Unmarshal(b, &cs.contexts); err != nil {
		return errors.Wrap(err, "error unmarshaling context map")
	}
	for k, ctx := range cs.contexts {
		if err := ctx.Validate(); err != nil {
			return errors.Wrapf(err, "error in context '%s'", k)
		}
		ctx.Name = k
	}
	return nil
}

func (cs *CtxState) initCurrent() error {
	currentCtxFile := CurrentContextFile()
	b, err := os.ReadFile(currentCtxFile)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errs.FileError(err, currentCtxFile)
	}

	var sc storedCurrent
	if err := json.Unmarshal(b, &sc); err != nil {
		return errors.Wrap(err, "error unmarshaling current context")
	}

	return cs.SetCurrent(sc.Context)
}

func (cs *CtxState) load() error {
	if cs.Enabled() && cs.GetCurrent() != nil {
		if err := cs.GetCurrent().Load(); err != nil {
			return fmt.Errorf("failed loading current context configuration: %w", err)
		}

		return nil
	}

	if err := cs.LoadVintage(""); err != nil {
		return fmt.Errorf("failed loading context configuration: %w", err)
	}

	return nil
}

// LoadVintage loads context configuration from the vintage (non-context) path.
func (cs *CtxState) LoadVintage(f string) error {
	if f == "" {
		f = DefaultsFile()
	}

	b, err := os.ReadFile(f)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errs.FileError(err, f)
	}

	cs.config = make(map[string]interface{})
	if err := json.Unmarshal(b, &cs.config); err != nil {
		return errors.Wrapf(err, "error parsing %s", f)
	}
	return nil
}

// GetConfig returns the current context configuration.
func (cs *CtxState) GetConfig() (map[string]interface{}, error) {
	if cs.Enabled() {
		cur := cs.GetCurrent()
		if cur == nil {
			return nil, errors.New("cannot get context configuration; no current context set")
		}
		return cur.config, nil
	}
	return cs.config, nil
}

// SetCurrent sets the current context or returns an error if a context
// with the given name does not exist.
func (cs *CtxState) SetCurrent(name string) error {
	var ok bool
	cs.current, ok = cs.contexts[name]
	if !ok {
		return errors.Errorf("could not load context '%s'", name)
	}
	if len(cs.current.config) == 0 {
		if err := cs.current.Load(); err != nil {
			return err
		}
	}
	return nil
}

type contextSelect struct {
	Name    string
	Context *Context
}

// PromptContext gets user input to select a context.
func (cs *CtxState) PromptContext() error {
	var items []*contextSelect
	for _, context := range cs.List() {
		items = append(items, &contextSelect{
			Name:    context.Name,
			Context: context,
		})
	}

	var ctxStr string
	if len(items) == 1 {
		if err := ui.PrintSelected("Context", items[0].Name); err != nil {
			return err
		}
		ctxStr = items[0].Name
	} else {
		i, _, err := ui.Select("Select a context for 'step':", items,
			ui.WithSelectTemplates(ui.NamedSelectTemplates("Context")))
		if err != nil {
			return err
		}
		ctxStr = items[i].Name
	}
	if err := cs.SetCurrent(ctxStr); err != nil {
		return err
	}
	return cs.SaveCurrent(ctxStr)
}

// Enabled returns true if one of the following is true:
//   - there is a current context configured
//   - the context map is (list of available contexts) is not empty.
func (cs *CtxState) Enabled() bool {
	return cs.current != nil || len(cs.contexts) > 0
}

// Contexts returns an object that enables context management.
func Contexts() *CtxState {
	return ctxState
}

// Add adds a new context to the context map. If current context is not
// set then store the new context as the current context for future commands.
func (cs *CtxState) Add(ctx *Context) error {
	if err := ctx.Validate(); err != nil {
		return errors.Wrapf(err, "error adding context")
	}
	if cs.contexts == nil {
		cs.contexts = map[string]*Context{ctx.Name: ctx}
	} else {
		cs.contexts[ctx.Name] = ctx
	}

	b, err := json.MarshalIndent(cs.contexts, "", "    ")
	if err != nil {
		return err
	}

	cf := ContextsFile()
	if err := os.MkdirAll(filepath.Dir(cf), 0700); err != nil {
		return errs.FileError(err, cf)
	}
	if err := os.WriteFile(cf, b, 0600); err != nil {
		return errs.FileError(err, cf)
	}

	if cs.current == nil {
		if err := cs.SaveCurrent(ctx.Name); err != nil {
			return err
		}
	}
	return nil
}

// GetCurrent returns the current context.
func (cs *CtxState) GetCurrent() *Context {
	return cs.current
}

// Get returns the context with the given name.
func (cs *CtxState) Get(name string) (ctx *Context, ok bool) {
	ctx, ok = cs.contexts[name]
	return
}

// Remove removes a context from the context state.
func (cs *CtxState) Remove(name string) error {
	if _, ok := cs.contexts[name]; !ok {
		return errors.Errorf("context '%s' not found", name)
	}
	if cs.current != nil && cs.current.Name == name {
		return errors.New("cannot remove current context; use 'step context select' to switch contexts")
	}

	delete(cs.contexts, name)

	b, err := json.MarshalIndent(cs.contexts, "", "    ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(ContextsFile(), b, 0600); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}
	return nil
}

// List returns a list of all contexts.
func (cs *CtxState) List() []*Context {
	l := make([]*Context, 0, len(cs.contexts))

	for _, v := range cs.contexts {
		l = append(l, v)
	}
	return l
}

// ListAlphabetical returns a case-insensitive, alphabetically
// sorted list of all contexts.
func (cs *CtxState) ListAlphabetical() []*Context {
	l := cs.List()

	sort.Slice(l, func(i, j int) bool {
		return strings.ToLower(l[i].Name) < strings.ToLower(l[j].Name)
	})

	return l
}

// SaveCurrent stores the given context name as the selected default context for
// future commands.
func (cs *CtxState) SaveCurrent(name string) error {
	if _, ok := Contexts().Get(name); !ok {
		return errors.Errorf("context '%s' not found", name)
	}

	b, err := json.Marshal(storedCurrent{Context: name})
	if err != nil {
		return err
	}
	//nolint:gosec // this file does not contain sensitive info
	if err = os.WriteFile(CurrentContextFile(), b, 0644); err != nil {
		return errs.FileError(err, CurrentContextFile())
	}
	return nil
}

// Apply the current context configuration to the command line environment.
func (cs *CtxState) Apply(ctx *cli.Context) error {
	cfg, err := cs.GetConfig()
	if err != nil {
		return err
	}
	for _, f := range ctx.Command.Flags {
		// Skip if EnvVar == IgnoreEnvVar
		if getFlagEnvVar(f) == IgnoreEnvVar {
			continue
		}

		for _, name := range strings.Split(f.GetName(), ",") {
			name = strings.TrimSpace(name)
			if ctx.IsSet(name) {
				break
			}
			// Set the flag for the first key that matches.
			if v, ok := cfg[name]; ok {
				ctx.Set(name, fmt.Sprintf("%v", v))
				break
			}
		}
	}
	return nil
}

// getEnvVar generates the environment variable for the given flag name.
func getEnvVar(name string) string {
	parts := strings.Split(name, ",")
	name = strings.TrimSpace(parts[0])
	name = strings.ReplaceAll(name, "-", "_")
	return "STEP_" + strings.ToUpper(name)
}

// getFlagEnvVar returns the value of the EnvVar field of a flag.
func getFlagEnvVar(f cli.Flag) string {
	v := reflect.ValueOf(f)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		envVar := v.FieldByName("EnvVar")
		if envVar.IsValid() {
			return envVar.String()
		}
	}
	return ""
}

// SetEnvVar sets the the EnvVar element to each flag recursively.
func SetEnvVar(c *cli.Command) {
	if c == nil {
		return
	}

	// Enable getting the flags from a json file
	if c.Before == nil && c.Action != nil {
		c.Before = getConfigVars
	}

	// Enable getting the flags from environment variables
	for i := range c.Flags {
		envVar := getEnvVar(c.Flags[i].GetName())
		switch f := c.Flags[i].(type) {
		case cli.BoolFlag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		case cli.BoolTFlag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		case cli.DurationFlag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		case cli.Float64Flag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		case cli.GenericFlag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		case cli.Int64Flag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		case cli.IntFlag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		case cli.IntSliceFlag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		case cli.Int64SliceFlag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		case cli.StringFlag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		case cli.StringSliceFlag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		case cli.Uint64Flag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		case cli.UintFlag:
			if f.EnvVar == "" {
				f.EnvVar = envVar
				c.Flags[i] = f
			}
		}
	}

	for i := range c.Subcommands {
		SetEnvVar(&c.Subcommands[i])
	}
}

// GetConfigVars load the defaults.json file and sets the flags if they are not
// already set or the EnvVar is set to IgnoreEnvVar.
//
// TODO(mariano): right now it only supports parameters at first level.
func getConfigVars(ctx *cli.Context) (err error) {
	if ctx.Bool("no-context") {
		return nil
	}

	cs := Contexts()

	// Load the the current context into memory:
	//  - If contexts enabled then make sure a current context is selected
	//    and loaded.
	//  - If vintage context then check if overwritten by --config flag.
	if cs.Enabled() {
		if ctx.IsSet("context") {
			err = cs.SetCurrent(ctx.String("context"))
		} else if cs.Enabled() && cs.GetCurrent() == nil {
			err = cs.PromptContext()
		}
		if err != nil {
			return err
		}
	} else if ctx.GlobalString("config") != "" {
		// Re-load the vintage context configuration if `--config` flag supplied.
		cs.LoadVintage(ctx.GlobalString("config"))
	}

	// TODO: a mock detail because of "add detail/assignee to this TODO/FIXME/BUG comment" lint issue
	fullCommandName := strings.ToLower(strings.TrimSpace(ctx.Command.FullName()))
	if strings.EqualFold(fullCommandName, "ca bootstrap-helper") {
		return nil
	}

	if err := cs.Apply(ctx); err != nil {
		return fmt.Errorf("error applying contexts: %w", err)
	}

	return nil
}
