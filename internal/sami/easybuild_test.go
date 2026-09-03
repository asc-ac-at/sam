package easybuild

import "testing"

func TestEbVersionFromPath(t *testing.T) {
	want := "5.3.0"
	got := EbVersionFromPath("asc_eb_5.3.0-nvidia-nvhpc-25.9.yaml")
	if got != want {
		t.Errorf(`versionFromPath(asc_eb_5.3.0-nvidia-nvhpc-25.9.yaml) = %q, want: %q`, got, want)
	}
}
