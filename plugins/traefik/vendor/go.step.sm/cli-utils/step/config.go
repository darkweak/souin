package step

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// PathEnv defines the name of the environment variable that can overwrite
// the default configuration path.
const PathEnv = "STEPPATH"

// HomeEnv defines the name of the environment variable that can overwrite the
// default home directory.
const HomeEnv = "HOME"

var (
	// version and buildTime are filled in during build by the Makefile
	name      = "Smallstep CLI"
	buildTime = "N/A"
	version   = "N/A"
)

var cache struct {
	sync.Once
	err          error
	stepBasePath string // stepBasePath will be populated in init() with the proper STEPPATH.
	homePath     string // homePath will be populated in init() with the proper HOME.
}

// Init initializes the step environment.
//
// A program calling this function should fail because the step variables would
// be undefined.
func Init() error {
	if err := initStepPath(); err != nil {
		return err
	}
	return Contexts().Init()
}

func initStepPath() error {
	cache.Do(func() {
		// Get home path from environment or from the user object.
		homePath := os.Getenv(HomeEnv)
		if homePath == "" {
			usr, err := user.Current()
			if err == nil && usr.HomeDir != "" {
				homePath = usr.HomeDir
			} else {
				cache.err = fmt.Errorf("error obtaining home directory, please define environment variable %s", HomeEnv)
				return
			}
		}

		// Get step path from environment or relative to home.
		stepBasePath := os.Getenv(PathEnv)
		if stepBasePath == "" {
			stepBasePath = filepath.Join(homePath, ".step")
		}

		// cleanup and add paths to cache
		cache.homePath = filepath.Clean(homePath)
		cache.stepBasePath = filepath.Clean(stepBasePath)
	})

	return cache.err
}

// Home returns the user home directory using the environment variable HOME or
// the os/user package.
func Home() string {
	_ = initStepPath()
	return cache.homePath
}

// BasePath returns the base path for the step configuration directory.
func BasePath() string {
	_ = initStepPath()
	return cache.stepBasePath
}

// Path returns the path for the step configuration directory.
//
//  1. If the base step path has a current context configured, then this method
//     returns the path to the authority configured in the context.
//  2. If the base step path does not have a current context configured this
//     method returns the value defined by the environment variable STEPPATH, OR
//  3. If no environment variable is set, this method returns `$HOME/.step`.
func Path() string {
	c := Contexts().GetCurrent()
	if c == nil {
		return BasePath()
	}
	return filepath.Join(BasePath(), "authorities", c.Authority)
}

// ProfilePath returns the path for the currently selected profile path.
//
//  1. If the base step path has a current context configured, then this method
//     returns the path to the profile configured in the context.
//  2. If the base step path does not have a current context configured this
//     method returns the value defined by the environment variable STEPPATH, OR
//  3. If no environment variable is set, this method returns `$HOME/.step`.
func ProfilePath() string {
	c := Contexts().GetCurrent()
	if c == nil {
		return BasePath()
	}
	return filepath.Join(BasePath(), "profiles", c.Profile)
}

// IdentityPath returns the location of the identity directory.
func IdentityPath() string {
	return filepath.Join(Path(), "identity")
}

// IdentityFile returns the location of the identity file.
func IdentityFile() string {
	return filepath.Join(Path(), "config", "identity.json")
}

// DefaultsFile returns the location of the defaults file at the base of the
// authority path.
func DefaultsFile() string {
	return filepath.Join(Path(), "config", "defaults.json")
}

// ProfileDefaultsFile returns the location of the defaults file at the base
// of the profile path.
func ProfileDefaultsFile() string {
	return filepath.Join(ProfilePath(), "config", "defaults.json")
}

// ConfigPath returns the location of the $(step path)/config directory.
func ConfigPath() string {
	return filepath.Join(Path(), "config")
}

// ProfileConfigPath returns the location of the $(step path --profile)/config directory.
func ProfileConfigPath() string {
	return filepath.Join(ProfilePath(), "config")
}

// CaConfigFile returns the location of the ca.json file -- configuration for
// connecting to the CA.
func CaConfigFile() string {
	return filepath.Join(Path(), "config", "ca.json")
}

// ContextsFile returns the location of the config file.
func ContextsFile() string {
	return filepath.Join(BasePath(), "contexts.json")
}

// CurrentContextFile returns the path to the file containing the current context.
func CurrentContextFile() string {
	return filepath.Join(BasePath(), "current-context.json")
}

// Abs returns the given path relative to the STEPPATH if it's not an
// absolute path, relative to the home directory using the special string "~/",
// or relative to the working directory using "./"
//
// Relative paths like 'certs/root_ca.crt' will be converted to
// '$STEPPATH/certs/root_ca.crt', but paths like './certs/root_ca.crt' will be
// relative to the current directory. Home relative paths like
// ~/certs/root_ca.crt will be converted to '$HOME/certs/root_ca.crt'. And
// absolute paths like '/certs/root_ca.crt' will remain the same.
func Abs(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	// Windows accept both \ and /
	slashed := filepath.ToSlash(path)
	switch {
	case strings.HasPrefix(slashed, "~/"):
		return filepath.Join(Home(), path[2:])
	case strings.HasPrefix(slashed, "./"), strings.HasPrefix(slashed, "../"):
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	default:
		return filepath.Join(Path(), path)
	}
}

// Set updates the name, version, and build time
func Set(n, v, t string) {
	name = n
	version = v
	buildTime = t
}

// Version returns the current version of the binary
func Version() string {
	out := version
	if version == "N/A" {
		out = "0000000-dev"
	}

	return fmt.Sprintf("%s/%s (%s/%s)",
		name, out, runtime.GOOS, runtime.GOARCH)
}

// ReleaseDate returns the time of when the binary was built
func ReleaseDate() string {
	out := buildTime
	if buildTime == "N/A" {
		out = time.Now().UTC().Format("2006-01-02 15:04 MST")
	}

	return out
}
