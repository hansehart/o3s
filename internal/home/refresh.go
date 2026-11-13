package home

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"
)

// FileChange is a file of the stack the binary ships in another shape than the home holds it.
type FileChange struct {
	// Path is where the file sits inside the stack folder.
	Path string
	// New marks a file the home lacks.
	New  bool
	data []byte
}

// homeKeys are the keys of the config that belong to the home, which a refresh keeps.
var homeKeys = []string{"features", "containerEnv"}

// Refresh compares the home's stack with the one the binary ships and hands back each file that differs.
func Refresh(dir string) ([]FileChange, error) {
	files, err := stack()
	// Hand back a stack the binary fails to hand over
	if err != nil {
		return nil, err
	}
	var changes []FileChange
	// Compare every file the binary ships
	for _, f := range files {
		have, err := os.ReadFile(filepath.Join(dir, stackDir, filepath.FromSlash(f.path)))
		// Name a file the home lacks
		if errors.Is(err, fs.ErrNotExist) {
			changes = append(changes, FileChange{Path: f.path, New: true, data: f.data})
			continue
		}
		// Hand back a file that fails to read
		if err != nil {
			return nil, err
		}
		want := f.data
		// Keep the home's own part of the config
		if f.path == "devcontainer.json" {
			want, err = withHomeKeys(want, have)
			// Hand back a config that fails to merge
			if err != nil {
				return nil, err
			}
		}
		// Name a file whose shape differs
		if !bytes.Equal(have, want) {
			changes = append(changes, FileChange{Path: f.path, data: want})
		}
	}
	return changes, nil
}

// Write brings each file a refresh named into the home's stack.
func Write(dir string, changes []FileChange) error {
	// Write every file that differs
	for _, c := range changes {
		// Hand back a file that stays unwritten
		if err := writeFile(filepath.Join(dir, stackDir, filepath.FromSlash(c.Path)), c.data); err != nil {
			return err
		}
	}
	return nil
}

// withHomeKeys hands back the binary's config with the home's features and env, as the home writes them.
func withHomeKeys(shipped, home []byte) ([]byte, error) {
	out, err := hujson.Parse(shipped)
	// Hand back a shipped config that holds no JSON with comments
	if err != nil {
		return nil, err
	}
	have, err := hujson.Parse(home)
	// Hand back a home config that holds no JSON with comments
	if err != nil {
		return nil, err
	}
	dst, ok1 := out.Value.(*hujson.Object)
	src, ok2 := have.Value.(*hujson.Object)
	// Take the binary's config as it is where either holds no object
	if !ok1 || !ok2 {
		return shipped, nil
	}
	// Carry every key the home owns across
	for _, key := range homeKeys {
		m, ok := member(src, key)
		// Leave a key the home lacks to the binary
		if !ok {
			continue
		}
		// Take the home's value in the binary's place, or add the home's member where the binary lacks it
		if i := index(dst, key); i >= 0 {
			dst.Members[i].Value = m.Value
		} else {
			dst.Members = append(dst.Members, m)
		}
	}
	return out.Pack(), nil
}

// member finds a member of an object by its name.
func member(o *hujson.Object, key string) (hujson.ObjectMember, bool) {
	i := index(o, key)
	// Report a member the object lacks
	if i < 0 {
		return hujson.ObjectMember{}, false
	}
	return o.Members[i], true
}

// index finds where a member of an object sits, and -1 where the object lacks it.
func index(o *hujson.Object, key string) int {
	// Look at every member's name
	for i, m := range o.Members {
		// Take the member of that name
		if l, ok := m.Name.Value.(hujson.Literal); ok && l.String() == key {
			return i
		}
	}
	return -1
}

// ImageChange is an o3s image whose tag an update moves.
type ImageChange struct {
	Name, Old, New string
}

// Images compares the tags of the o3s images in the home's compose with the ones this binary writes.
func Images(dir string) ([]ImageChange, error) {
	have, err := os.ReadFile(filepath.Join(dir, stackDir, "compose.yaml"))
	// Name nothing for a home without a compose file, which a refresh writes in full
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	// Hand back a compose file that fails to read
	if err != nil {
		return nil, err
	}
	want, err := files.ReadFile(source + "/compose.yaml")
	// Hand back a compose file the binary fails to hand over
	if err != nil {
		return nil, err
	}
	want = pinned(want)
	var changes []ImageChange
	// Compare every o3s image's tag
	for _, image := range images {
		old, next := tagOf(have, image), tagOf(want, image)
		// Name an image whose tag moves
		if old != next {
			changes = append(changes, ImageChange{Name: path.Base(image), Old: old, New: next})
		}
	}
	return changes, nil
}

// tagOf reads the tag a compose file gives an image, without its digest.
func tagOf(compose []byte, image string) string {
	// Look at every line for the image
	for line := range strings.Lines(string(compose)) {
		_, rest, ok := strings.Cut(line, "image: "+image+":")
		// Take the tag up to the digest or the end of the line
		if ok {
			tag, _, _ := strings.Cut(strings.TrimSpace(rest), "@")
			return tag
		}
	}
	return ""
}
