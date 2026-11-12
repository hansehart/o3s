package release

import (
	"os"
	"strings"
	"testing"
)

// The workflow every binary accepts as the signer lives in the repo, so a rename fails here before it fails every update.
func TestTheSignerWorkflowExists(t *testing.T) {
	path := "../../" + strings.TrimPrefix(signer, repo+"/")

	_, err := os.Stat(path)

	// Report a signer the repo lacks
	if err != nil {
		t.Errorf("signer %s: %v", signer, err)
	}
}
