package walker

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/QubeUtils/repowalk/internal/ignore"
	"github.com/pkoukk/tiktoken-go"
	"golang.org/x/sync/errgroup"
)

type FileNode struct {
	Path      string
	Name      string
	IsDir     bool
	Size      int64
	Content   []byte
	IsBinary  bool
	IsIgnored bool
	Tokens    int

	OriginalLines []string
	LineStates    map[int]int // 0: Normal, 1: Excluded, 2: Exclusively Included
}

// FileSystem is an interface to allow mocking file system operations for testing.
type FileSystem interface {
	WalkDir(root string, fn fs.WalkDirFunc) error
	ReadFile(name string) ([]byte, error)
	Stat(name string) (fs.FileInfo, error)
}

// OSFileSystem implements FileSystem using the real os/filepath package.
type OSFileSystem struct{}

func (OSFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func (OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (OSFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// WalkOptions configures the behavior of the traversal engine.
type WalkOptions struct {
	RootPath      string
	MaxSize       int64
	Microservices []string
	MaxWorkers    int
	RedactSecrets bool
	IgnoreExts    []string
}

type Walker struct {
	fs            FileSystem
	ignore        ignore.Matcher
	opts          WalkOptions
	tkm           *tiktoken.Tiktoken
	ignoreExtsMap map[string]bool
}

// NewWalker creates a new Walker.
func NewWalker(fs FileSystem, matcher ignore.Matcher, opts WalkOptions) *Walker {
	if opts.MaxWorkers <= 0 {
		opts.MaxWorkers = 10 // Default worker pool size
	}
	tkm, _ := tiktoken.GetEncoding("cl100k_base") // Ignore error for now
	ignoreExtsMap := make(map[string]bool)
	for _, ext := range opts.IgnoreExts {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		ignoreExtsMap[ext] = true
	}

	return &Walker{
		fs:            fs,
		ignore:        matcher,
		opts:          opts,
		tkm:           tkm,
		ignoreExtsMap: ignoreExtsMap,
	}
}

func isBinary(content []byte) bool {
	limit := 512
	if len(content) < limit {
		limit = len(content)
	}
	for i := 0; i < limit; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

var secretRegex = regexp.MustCompile(`(?i)(api_key|apikey|secret|token|password)[^a-zA-Z0-9]{1,5}['"][a-zA-Z0-9\-_]{16,}['"]`)

func redactSecrets(content []byte) []byte {
	return secretRegex.ReplaceAllFunc(content, func(match []byte) []byte {
		// Replace the actual secret string but keep the key
		valRegex := regexp.MustCompile(`['"][a-zA-Z0-9\-_]{16,}['"]`)
		loc := valRegex.FindIndex(match)
		if loc != nil {
			res := append([]byte{}, match[:loc[0]]...)
			res = append(res, []byte(`"[REDACTED_SECRET]"`)...)
			return res
		}
		return match
	})
}

// Walk traverses the file system concurrently and returns the collected FileNodes.
func (w *Walker) Walk(ctx context.Context) ([]FileNode, error) {
	var nodes []FileNode
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)

	// Channel to queue files for processing by workers
	type workItem struct {
		Path      string
		Name      string
		Info      fs.FileInfo
		IsIgnored bool
	}
	workChan := make(chan workItem, 1000)

	// Start worker pool
	for i := 0; i < w.opts.MaxWorkers; i++ {
		g.Go(func() error {
			for item := range workChan {
				if ctx.Err() != nil {
					return ctx.Err()
				}

				node := FileNode{
					Path:      item.Path,
					Name:      item.Name,
					IsDir:     item.Info.IsDir(),
					Size:      item.Info.Size(),
					IsIgnored: item.IsIgnored,
				}

				// Read file content if within limits and not ignored or dir
				if !node.IsIgnored && !node.IsDir {
					if node.Size <= w.opts.MaxSize {
						content, err := w.fs.ReadFile(item.Path)
						if err == nil {
							node.IsBinary = isBinary(content)
							if !node.IsBinary {
								if w.opts.RedactSecrets {
									content = redactSecrets(content)
								}
								node.Content = content
								if w.tkm != nil {
									node.Tokens = len(w.tkm.Encode(string(content), nil, nil))
								}
							}
						}
					} else {
						// Too large, skip content
						node.IsBinary = true
					}
				}

				mu.Lock()
				nodes = append(nodes, node)
				mu.Unlock()
			}
			return nil
		})
	}

	// Single goroutine to traverse the directory tree and feed the work channel
	g.Go(func() error {
		defer close(workChan) // Close channel when traversal is done

		return w.fs.WalkDir(w.opts.RootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err // Return traversal error
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Get relative path for ignore matching
			relPath, err := filepath.Rel(w.opts.RootPath, path)
			if err != nil || relPath == "." || relPath == "" {
				// Skip emitting the root directory itself to avoid duplicating the root node in the UI
				if d.IsDir() && relPath == "." {
					return nil
				}
			} else {
				// Normalize to forward slashes for ignore matcher
				relPath = filepath.ToSlash(relPath)

				if w.ignore != nil && w.ignore.MatchesPath(relPath) {
					info, err := d.Info()
					if err == nil {
						select {
						case workChan <- workItem{Path: path, Name: d.Name(), Info: info, IsIgnored: true}:
						case <-ctx.Done():
							return ctx.Err()
						}
					}

					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				} else if !d.IsDir() && w.ignoreExtsMap[strings.ToLower(filepath.Ext(d.Name()))] {
					info, err := d.Info()
					if err == nil {
						select {
						case workChan <- workItem{Path: path, Name: d.Name(), Info: info, IsIgnored: true}:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
					return nil
				}

				// If microservices are specified, we only traverse the root,
				// those specified directories, and their children.
				if len(w.opts.Microservices) > 0 {
					allowed := false
					for _, ms := range w.opts.Microservices {
						msNormalized := filepath.ToSlash(ms)
						if relPath == msNormalized || strings.HasPrefix(relPath, msNormalized+"/") || strings.HasPrefix(msNormalized, relPath+"/") {
							allowed = true
							break
						}
					}
					if !allowed {
						if d.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
				}
			}

			info, err := d.Info()
			if err != nil {
				return nil
			}

			select {
			case workChan <- workItem{Path: path, Name: d.Name(), Info: info, IsIgnored: false}:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return nodes, nil
}
