package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIndexContainsInteractiveDashboardControls(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	(&Server{}).index(rr, req)
	body := rr.Body.String()
	for _, want := range []string{
		"<canvas id=\"chart\"",
		"id=\"range\"",
		"id=\"resetZoom\"",
		"MMR per Hero",
		"Auto sync",
		"sync --auto",
		"window.devicePixelRatio",
		"hoverInfo",
		"pointerLine",
		"Zoom selection",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index missing %q", want)
		}
	}
	
}
