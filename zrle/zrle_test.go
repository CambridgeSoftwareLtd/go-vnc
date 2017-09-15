package zrle

import (
	"bytes"
	"reflect"
	"testing"
)

func TestTilesToPixels(t *testing.T) {
	Tiles := []Tile{
		Tile{X: 0, Y: 0, Width: 1, Height: 1, Pixels: []CPixel{CPixel{0}}},
		Tile{X: 1, Y: 0, Width: 2, Height: 1, Pixels: []CPixel{CPixel{1}, CPixel{2}}},
	}
	pixels := TilesToPixels(3, 1, Tiles)
	expected := [][]CPixel{[]CPixel{CPixel{0}, CPixel{1}, CPixel{2}}}
	if !reflect.DeepEqual(expected, pixels) {
		t.Errorf("expected %v, got %v", expected, pixels)
	}
}

func TestTileToPixelGrid_GridCase(t *testing.T) {
	tile := Tile{
		Width:  2,
		Height: 2,
		Pixels: []CPixel{
			CPixel{0}, CPixel{1}, CPixel{2}, CPixel{3},
		},
	}
	pixels := tile.ToPixelGrid()
	expected := [][]CPixel{
		[]CPixel{CPixel{0}, CPixel{1}},
		[]CPixel{CPixel{2}, CPixel{3}},
	}
	if !reflect.DeepEqual(expected, pixels) {
		t.Errorf("expected %v, got %v", expected, pixels)
	}
}

func TestTileToPixelGrid_ColumnCase(t *testing.T) {
	tile := Tile{
		Width:  4,
		Height: 1,
		Pixels: []CPixel{
			CPixel{0}, CPixel{1}, CPixel{2}, CPixel{3},
		},
	}
	pixels := tile.ToPixelGrid()
	expected := [][]CPixel{
		[]CPixel{CPixel{0}, CPixel{1}, CPixel{2}, CPixel{3}},
	}
	if !reflect.DeepEqual(expected, pixels) {
		t.Errorf("expected %v, got %v", expected, pixels)
	}
}

func TestZRLEncoding_CalcRuns(t *testing.T) {
	buf := bytes.NewReader([]byte{0})

	result, n := CalcRuns(buf, 255)
	// Check result
	exp := 1
	if result != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check bytes read
	exp = 1
	if n != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check buffer is consumed
	exp = 0
	if buf.Len() != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}

	buf = bytes.NewReader([]byte{254})
	result, n = CalcRuns(buf, 255)
	// Check result
	exp = 255
	if result != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check bytes read
	exp = 1
	if n != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check buffer is consumed
	exp = 0
	if buf.Len() != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}

	buf = bytes.NewReader([]byte{255, 0})
	result, n = CalcRuns(buf, 255)
	// Check result
	exp = 256
	if result != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check bytes read
	exp = 2
	if n != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check buffer is consumed
	exp = 0
	if buf.Len() != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}

	buf = bytes.NewReader([]byte{255, 1})
	result, n = CalcRuns(buf, 255)
	// Check result
	exp = 257
	if result != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check bytes read
	exp = n
	if n != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check buffer is consumed
	exp = 0
	if buf.Len() != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}

	buf = bytes.NewReader([]byte{255, 254})
	result, n = CalcRuns(buf, 255)
	// Check result
	exp = 510
	if result != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check bytes read
	exp = 2
	if n != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check buffer is consumed
	exp = 0
	if buf.Len() != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}

	buf = bytes.NewReader([]byte{255, 255, 0})
	result, n = CalcRuns(buf, 255)
	// Check result
	exp = 511
	if result != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check bytes read
	exp = 3
	if n != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check buffer is consumed
	exp = 0
	if buf.Len() != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}

	// Check read STOPS correctly

	buf = bytes.NewReader([]byte{255, 255, 0, 255})
	result, n = CalcRuns(buf, 255)
	// Check result
	exp = 511
	if result != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check bytes read
	exp = 3
	if n != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}
	// Check buffer is consumed
	exp = 1
	if buf.Len() != exp {
		t.Errorf("expected %v, got %v", exp, result)
	}

}
