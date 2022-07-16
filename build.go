//go:build ignore

package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	tags     = ""
	testTags = "test"
)

var programs = map[string][]*exec.Cmd{}

var lastRun time.Time

type flagStrs []string

func (s *flagStrs) String() string {
	var sb strings.Builder

	for _, str := range *s {
		sb.WriteString(str)
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String())
}

func (s *flagStrs) Set(value string) error {
	*s = append(*s, value)

	return nil
}

var opts struct {
	goos              string
	goarch            string
	tags              string
	testTags          string
	unoptimised       bool
	debug             bool
	race              bool
	build             bool
	clear             bool
	cover             bool
	generate          bool
	vet               bool
	test              bool
	verbose           bool
	cmdStrs           flagStrs
	watch             bool
	watchExts         string
	watchSkipPatterns string
	watchInterval     time.Duration
}

func main() {
	flag.StringVar(&opts.goos, "goos", "", "Sets the GOOS environment variable for the build")
	flag.StringVar(&opts.goarch, "goarch", "", "Sets the GOARCH environment variable for the build")
	flag.StringVar(&opts.tags, "tags", "", "Additional build tags")
	flag.StringVar(&opts.testTags, "test-tags", "", "Additional test build tags")
	flag.BoolVar(&opts.unoptimised, "unoptimised", false, "Disable optimisations/inlining")
	flag.BoolVar(&opts.debug, "debug", false, "Enable symbol table/DWARF generation")
	flag.BoolVar(&opts.race, "race", false, "Enable data race detection in the final binary")
	flag.BoolVar(&opts.build, "build", false, "Enable building the final binary")
	flag.BoolVar(&opts.clear, "clear", false, "If set then the console will be cleared before running the build")
	flag.BoolVar(&opts.cover, "cover", false, "Generates an HTML cover report that's opened in the browser if watch is disabled")
	flag.BoolVar(&opts.generate, "generate", false, "Enable generating code before build")
	flag.BoolVar(&opts.vet, "vet", false, "Enable vet before build")
	flag.BoolVar(&opts.test, "test", false, "Enable tests before build")
	flag.BoolVar(&opts.verbose, "verbose", false, "Print the commands that are being run along with all command output")
	flag.Var(&opts.cmdStrs, "start", "Commands to run after building programs in watch mode")
	flag.BoolVar(&opts.watch, "watch", false, "Watches for changes and re-runs the build if changes are detected")
	flag.StringVar(&opts.watchExts, "watch-exts", ".go .h .c .sql .json", "A space separated list of file extensions to watch")
	flag.StringVar(&opts.watchSkipPatterns, "watch-skip-patterns", ".git/ .hg/ .svn/ node_modules/ build.go", "A space separated list of patterns to skip in watch mode")
	flag.DurationVar(&opts.watchInterval, "watch-interval", 2*time.Second, "The interval that watch mode checks for file changes")
	flag.Parse()

	if s := strings.TrimSpace(opts.tags); s != "" {
		tags += " " + s
	}
	tags = strings.TrimSpace(tags)

	if s := strings.TrimSpace(opts.testTags); s != "" {
		testTags += " " + s
	}
	testTags = strings.TrimSpace(tags + " " + testTags)

	opts.watchExts = strings.TrimSpace(opts.watchExts)
	opts.watchSkipPatterns = strings.TrimSpace(opts.watchSkipPatterns)

	if !opts.build && !opts.generate && !opts.vet && !opts.test && !opts.cover {
		opts.generate = true
		opts.vet = true
		opts.test = true
		opts.build = true
	}

	mainPackages, err := packages()
	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	pkgs := flag.Args()
	if len(pkgs) == 0 {
		pkgs = mainPackages
	} else {
		for i, pkg := range pkgs {
			// We only want to replace packages that are relative paths
			if !strings.HasPrefix(pkg, "./") {
				continue
			}

			// If any of the packages just contain ./... then we want to build
			// all packages anyway, so we break out here
			if pkg == "./..." {
				pkgs = mainPackages

				break
			}

			pkg = strings.TrimPrefix(pkg, "./")
			pkg = strings.TrimSuffix(pkg, "/...")
			pkg = strings.TrimSuffix(pkg, "/")

			for _, mainPkg := range mainPackages {
				if strings.Contains(mainPkg, pkg) {
					pkgs[i] = mainPkg
				}
			}
		}
	}

	if opts.goos == "" {
		opts.goos = runtime.GOOS
	}

	if opts.goarch == "" {
		opts.goarch = runtime.GOARCH
	}

	// Always immediately run the build pipeline at least once, even if in watch mode
	run(pkgs)

	if opts.watch {
		fmt.Println("-> watching for changes...")

		skipPatterns := strings.Fields(opts.watchSkipPatterns)
		skip := func(path string) bool {
			for _, pattern := range skipPatterns {
				path = filepath.ToSlash(path)

				if path == pattern {
					return true
				}

				if strings.HasSuffix(pattern, "/") && strings.HasPrefix(path, pattern) {
					return true
				}
			}

			return false
		}

		exts := make(map[string]struct{})
		for _, ext := range strings.Fields(opts.watchExts) {
			if ext == "" {
				continue
			}

			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}

			exts[ext] = struct{}{}
		}

		files := make(map[string]time.Time)
		for {
			var shouldRun bool

			_ = filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Completely skip directories that are in the skip patterns
				if entry.IsDir() && skip(path) {
					return filepath.SkipDir
				}

				// Individually skip directories/files that haven't been entirely skipped by the previous check
				if entry.IsDir() || skip(path) {
					return nil
				}

				// Skip any files that don't match watch extensions
				if _, ok := exts[filepath.Ext(path)]; !ok {
					return nil
				}

				fi, err := entry.Info()
				if err != nil {
					return err
				}

				if modified, ok := files[path]; !shouldRun && ok {
					shouldRun = modified.Before(fi.ModTime()) && lastRun.Before(fi.ModTime())
				}

				files[path] = fi.ModTime()

				return nil
			})

			if shouldRun {
				run(pkgs)

				fmt.Println("-> watching for changes...")
			}

			time.Sleep(opts.watchInterval)
		}
	}
}

func run(pkgs []string) {
	lastRun = time.Now()

	if opts.clear {
		clear()
	}

	fmt.Print("-> running build:\n")

	for _, pkg := range pkgs {
		kill(pkg)
	}

	if opts.generate {
		if err := generate(); err != nil {
			if !opts.watch {
				os.Exit(1)
			}

			return
		}
	}

	if opts.vet {
		if err := vet(); err != nil {
			if !opts.watch {
				os.Exit(1)
			}

			return
		}
	}

	// If cover is enabled then we skip this because cover includes a call to test with cover profile flags anyway
	if opts.test && !opts.cover {
		if err := test(); err != nil {
			if !opts.watch {
				os.Exit(1)
			}

			return
		}
	}

	if opts.cover {
		if err := cover(); err != nil {
			if !opts.watch {
				os.Exit(1)
			}

			return
		}
	}

	if opts.build {
		for _, pkg := range pkgs {
			if err := build(pkg); err != nil {
				if !opts.watch {
					os.Exit(1)
				}

				continue
			}

			start(pkg)
		}
	}
}

func clear() {
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()

	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func command(program string, args ...string) (string, []string, string) {
	messageValues := make([]any, len(args))
	for i, arg := range args {
		messageValues[i] = arg
	}

	verbs := make([]string, len(args))
	for i, arg := range args {
		if strings.IndexFunc(arg, unicode.IsSpace) >= 0 {
			verbs[i] = "%q"
		} else {
			verbs[i] = "%v"
		}
	}
	message := fmt.Sprintf("%v "+strings.Join(verbs, " "), append([]any{program}, messageValues...)...)

	return program, args, message
}

func packages() ([]string, error) {
	out, err := exec.Command("go", "list", "-f", "[{{ .Name }}]{{ .ImportPath }}", "./...").CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			fmt.Println(string(out))
		}

		return nil, err
	}

	var packages []string
	for _, pkg := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(pkg, "[main]") {
			continue
		}

		packages = append(packages, strings.TrimPrefix(pkg, "[main]"))
	}

	return packages, nil
}

func generate() error {
	program, args, message := command("go", "generate", "./...")
	prefix := "->"
	if opts.verbose {
		prefix = "  "
		fmt.Printf("-> %v\n", message)
	}

	fmt.Printf("%v go generate... ", prefix)

	if out, err := exec.Command(program, args...).CombinedOutput(); err != nil {
		fmt.Println("error")

		if len(out) > 0 {
			fmt.Println(string(out))
		}

		return err
	} else {
		fmt.Println("ok")

		if len(out) > 0 && opts.verbose {
			fmt.Println(string(out))
		}
	}

	return nil
}

func test() error {
	program, args, message := command("go", "test", "-race", "-tags", testTags, "./...")
	prefix := "->"
	if opts.verbose {
		prefix = "  "
		fmt.Printf("-> %v\n", message)
	}

	fmt.Printf("%v go test... ", prefix)

	if out, err := exec.Command(program, args...).CombinedOutput(); err != nil {
		fmt.Println("error")

		if len(out) > 0 {
			fmt.Println(string(out))
		}

		return err
	} else {
		fmt.Println("ok")

		if len(out) > 0 && opts.verbose {
			fmt.Println(string(out))
		}
	}

	return nil
}

func cover() error {
	program, args, message := command("go", "test", "-race", "-tags", testTags, "-coverprofile", "_cover.out", "./...")
	prefix := "->"
	if opts.verbose {
		prefix = "  "
		fmt.Printf("-> %v\n", message)
	}

	fmt.Printf("%v go test (cover)... ", prefix)

	if out, err := exec.Command(program, args...).CombinedOutput(); err != nil {
		fmt.Println("error")

		if len(out) > 0 {
			fmt.Println(string(out))
		}

		return err
	} else {
		fmt.Println("ok")

		if len(out) > 0 && opts.verbose {
			fmt.Println(string(out))
		}
	}

	if !opts.watch {
		program, args, message := command("go", "tool", "cover", "-html", "_cover.out")
		prefix := "->"
		if opts.verbose {
			prefix = "  "
			fmt.Printf("-> %v\n", message)
		}

		fmt.Printf("%v go tool cover... ", prefix)

		if out, err := exec.Command(program, args...).CombinedOutput(); err != nil {
			fmt.Println("error")

			if len(out) > 0 {
				fmt.Println(string(out))
			}

			return err
		} else {
			fmt.Println("ok")

			if len(out) > 0 && opts.verbose {
				fmt.Println(string(out))
			}
		}
	}

	return nil
}

func vet() error {
	program, args, message := command("go", "vet", "./...")
	prefix := "->"
	if opts.verbose {
		prefix = "  "
		fmt.Printf("-> %v\n", message)
	}

	fmt.Printf("%v go vet... ", prefix)

	if out, err := exec.Command(program, args...).CombinedOutput(); err != nil {
		fmt.Println("error")

		if len(out) > 0 {
			fmt.Println(string(out))
		}

		return err
	} else {
		fmt.Println("ok")

		if len(out) > 0 && opts.verbose {
			fmt.Println(string(out))
		}
	}

	return nil
}

func build(pkg string) error {
	tagsMessage := tags
	if tagsMessage == "" {
		tagsMessage = "-"
	}

	args := []string{"build", "-v", "-x", "-o", ".", "-tags", tags}
	gcflags := []string{}
	ldflags := []string{}

	if opts.unoptimised {
		// -N disables all optimisations
		// -l disables inlining
		// See: go tool compile --help
		gcflags = append(gcflags, "all=-N -l")
	}

	if opts.debug {
		if opts.goos == "windows" {
			// This is required on Windows to view disassembly in things like pprof
			args = append(args, "-buildmode", "exe")
		}
	} else {
		args = append(args, "-trimpath")

		// -s disables the symbol table
		// -w disables DWARF generation
		// See: go tool link --help
		ldflags = append(ldflags, "-s")
		ldflags = append(ldflags, "-w")
	}

	if opts.race {
		args = append(args, "-race")
	}

	if len(gcflags) > 0 {
		args = append(args, "-gcflags", strings.Join(gcflags, " "))
	}

	if len(ldflags) > 0 {
		args = append(args, "-ldflags", strings.Join(ldflags, " "))
	}

	args = append(args, pkg)

	var env []string
	if opts.goos != "" {
		env = append(env, "GOOS="+opts.goos)
	}
	if opts.goarch != "" {
		env = append(env, "GOARCH="+opts.goarch)
	}

	program, args, message := command("go", args...)
	prefix := "->"
	if opts.verbose {
		prefix = "  "
		fmt.Printf("-> %v\n", message)
	}

	fmt.Printf("%v go build %v... ", prefix, strings.TrimSuffix(pkg, "..."))

	cmd := exec.Command(program, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Println("error")

		if len(out) > 0 {
			fmt.Println(string(out))
		}

		return err
	} else {
		fmt.Print("ok ")

		var info []string

		if opts.debug {
			info = append(info, "debug")
		} else {
			info = append(info, "release")
		}

		if opts.race {
			info = append(info, "race")
		}

		fmt.Printf("(%v)\n", strings.Join(info, "/"))

		if len(out) > 0 && opts.verbose {
			fmt.Println(string(out))
		}
	}

	return nil
}

func kill(pkg string) {
	for i, cmd := range programs[pkg] {
		if runtime.GOOS == "windows" {
			pid := strconv.Itoa(cmd.Process.Pid)
			out, err := exec.Command("tasklist", "/fi", "pid eq "+pid).CombinedOutput()
			if err != nil {
				fmt.Println(err)
			}

			if processRunning := !strings.Contains(strings.ToLower(string(out)), "no tasks"); !processRunning {
				continue
			}

			fmt.Printf("-> killing %v process #%v...\n", pkg, i)

			if err := cmd.Process.Kill(); err != nil {
				fmt.Println(err)
				fmt.Printf("-> forcibly killing %v process #%v...\n", pkg, i)

				if err := exec.Command("taskkill", "/pid", pid, "/f").Run(); err != nil {
					fmt.Println(err)

					continue
				}
			}
		} else {
			fmt.Printf("-> interrupting %v process #%v...\n", pkg, i)

			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				fmt.Println(err)
				fmt.Printf("-> killing %v process #%v...\n", pkg, i)

				if err := cmd.Process.Kill(); err != nil {
					fmt.Println(err)
				}

				continue
			}

			if _, err := cmd.Process.Wait(); err != nil {
				fmt.Println(err)
				fmt.Printf("-> killing %v process #%v...\n", pkg, i)

				if err := cmd.Process.Kill(); err != nil {
					fmt.Println(err)
				}

				continue
			}
		}
	}

	programs[pkg] = nil
}

func start(pkg string) {
	parts := strings.Split(pkg, "/")
	binaryName := parts[len(parts)-1]

	for i, cmdStr := range opts.cmdStrs {
		fields := strings.Fields(cmdStr)
		if fields[0] != binaryName {
			continue
		}

		program, args, message := command(binaryName, fields[1:]...)

		fmt.Printf("-> starting #%v... %v\n", i, message)

		cmd := exec.Command(program, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		programs[pkg] = append(programs[pkg], cmd)

		if err := cmd.Start(); err != nil {
			fmt.Println(err)

			continue
		}
	}
}
