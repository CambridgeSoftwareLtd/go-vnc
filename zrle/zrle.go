package zrle

import (
	"fmt"
	"io"
	"log"
)

// CPixel defines the structure of a CPIXEL
type CPixel []byte

// SubType defines the structure of a ZLRE sub-encoding type
type SubType uint8

const (
	raw SubType = iota
	solid
	packedPalette
	rle
	prle
)

const (
	// TileWidth is the expected, standard width of a tile
	TileWidth int = 64
	// TileHeight is the expected, standard height of a tile
	TileHeight = 64
)

// TileConfig defines the underlying structure of all tiles
type TileConfig struct {
	width, height int
}

// Tile defines a tile
type Tile struct {
	X, Y, Width, Height, BytesPerCPixel, SubType int
	Pixels                                       []CPixel
}

func (t Tile) String() string {
	return fmt.Sprintf("\n{ X: %v, Y: %v, Width: %v, Height: %v, BytesPerCPixel: %v, SubType: %v, CPIXELS: %v}\n", t.X, t.Y, t.Width, t.Height, t.BytesPerCPixel, t.SubType, len(t.Pixels))
}

// ToPixelGrid converts a tile into a CPIXEL grid
func (t Tile) ToPixelGrid() [][]CPixel {
	pixels := make([][]CPixel, t.Height)
	count := 0

	for i := range pixels {
		pixels[i] = make([]CPixel, t.Width)
	}

	x, y := 0, 0
	for _, pixel := range t.Pixels {
		pixels[y][x] = pixel
		x++
		count++
		if x == t.Width {
			y++
			x = 0
		}
	}
	return pixels
}

// CreateTiles creates a grid of tiles based on a width and height
func CreateTiles(width int, height int) (tiles []Tile) {
	x, y := 0, 0
	for height > 0 {
		rowWidth := width

		// If row is shorter than TileHeight adjust
		rowHeight := TileHeight
		if height < rowHeight {
			rowHeight = height
		}
		height -= rowHeight

		for rowWidth > 0 {

			// If tile is narrower than TileWidth adjust
			tileWidth := TileWidth
			if rowWidth < tileWidth {
				tileWidth = rowWidth
			}
			rowWidth -= tileWidth

			newTile := Tile{X: x, Y: y, Width: tileWidth, Height: rowHeight}
			tiles = append(tiles, newTile)

			x += tileWidth
		}
		x = 0
		y += rowHeight
	}
	return
}

// TilesToPixels converts a grid of tiles to CPIXEL data
func TilesToPixels(width int, height int, tiles []Tile) [][]CPixel {
	pixels := make([][]CPixel, height)
	for i := range pixels {
		pixels[i] = make([]CPixel, width)
	}
	for _, tile := range tiles {
		tilePixels := tile.ToPixelGrid()
		for i, tileRow := range tilePixels {
			for j, pixel := range tileRow {
				pixels[tile.Y+i][tile.X+j] = pixel
			}
		}
	}
	return pixels
}

// Subencoding defines a subencoding structure
type Subencoding interface {
	SubType() SubType
	Read(buf io.Reader, t *Tile) (int, error)
	String() string
}

type RawEncoding struct{}
type SolidEncoding struct{}
type PackedPaletteEncoding struct{}
type RleEncoding struct{}
type PrleEncoding struct{}

func (RawEncoding) SubType() SubType           { return raw }
func (SolidEncoding) SubType() SubType         { return solid }
func (PackedPaletteEncoding) SubType() SubType { return packedPalette }
func (RleEncoding) SubType() SubType           { return rle }
func (PrleEncoding) SubType() SubType          { return prle }

func (RawEncoding) String() string           { return "RawEncoding" }
func (SolidEncoding) String() string         { return "SolidEncoding" }
func (PackedPaletteEncoding) String() string { return "PackedPaletteEncoding" }
func (RleEncoding) String() string           { return "RleEncoding" }
func (PrleEncoding) String() string          { return "PrleEncoding" }

func (RawEncoding) Read(buf io.Reader, t *Tile) (bytesRead int, err error) {
	log.Printf("  Raw Subencoding - %v x %v", t.Width, t.Height)

	for i := 0; i < t.Width*t.Height; i++ {
		pixel := make(CPixel, t.BytesPerCPixel)
		n, err := io.ReadAtLeast(buf, pixel, t.BytesPerCPixel)
		if err != nil {
			return bytesRead, err
		}
		bytesRead += n

		if err != nil {
			return bytesRead, err
		}

		t.Pixels = append(t.Pixels, pixel)
	}

	return bytesRead, nil
}

func (SolidEncoding) Read(buf io.Reader, t *Tile) (int, error) {
	log.Println("  Solid Subencoding")

	pixel := make(CPixel, t.BytesPerCPixel)
	n, err := io.ReadAtLeast(buf, pixel, len(pixel))
	if err != nil {
		return n, err
	}
	for i := 0; i < (t.Width * t.Height); i++ {
		t.Pixels = append(t.Pixels, pixel)
	}

	return n, nil
}

func (PackedPaletteEncoding) Read(buf io.Reader, t *Tile) (int, error) {
	log.Println("  Packed Palette Subencoding")
	bytesToRead := t.SubType * t.BytesPerCPixel
	bytesRead := 0

	palette := make([]CPixel, 0)

	for bytesToRead > 0 {
		pixel := make(CPixel, t.BytesPerCPixel)
		n, err := io.ReadAtLeast(buf, pixel, len(pixel))
		bytesRead += n
		if err != nil {
			return bytesRead, err
		}
		palette = append(palette, pixel)
		bytesToRead -= len(pixel)
	}

	px := uint8(0)

	switch {
	case t.SubType == 2:
		bytesToRead = ((t.Width + 7) / 8) * t.Height
		px = uint8(1)
	case t.SubType >= 5:
		bytesToRead = ((t.Width + 1) / 2) * t.Height
		px = uint8(4)
	default:
		bytesToRead = ((t.Width + 3) / 4) * t.Height
		px = uint8(2)
	}

	for y := 0; y < t.Height; y++ {
		bx := make([]byte, 1)
		var b uint8

		nb := uint8(0)

		for x := 0; x < t.Width; x++ {
			if nb == 0 {
				n, err := io.ReadAtLeast(buf, bx, 1)
				b = uint8(bx[0])
				nb = uint8(8)

				if err != nil {
					return bytesRead, err
				}
				bytesRead += n

			}

			nb -= px
			idx := (b >> nb) & ((1 << px) - 1) & 127
			pixel := palette[idx]

			t.Pixels = append(t.Pixels, pixel)
		}
	}

	log.Print(t.Width*t.Height, len(t.Pixels))
	return bytesRead, nil
}

func (RleEncoding) Read(buf io.Reader, t *Tile) (int, error) {
	log.Println("  RLE Subecoding")

	bytesRead := 0
	pixelsRead := 0

	for pixelsRead < int(t.Width)*int(t.Height) {
		pixel := make(CPixel, t.BytesPerCPixel)
		n, err := io.ReadAtLeast(buf, pixel, len(pixel))
		bytesRead += n
		if err != nil {
			return bytesRead, err
		}

		runLength, n := CalcRuns(buf, 255)
		bytesRead += n
		rSize := (runLength - 1) / 255
		for i := 0; i < rSize; i++ {
			t.Pixels = append(t.Pixels, pixel)
			pixelsRead++
		}
	}

	return bytesRead, nil
}

func (PrleEncoding) Read(buf io.Reader, t *Tile) (int, error) {
	log.Println("  PRLE Subencoding")

	paletteSize := (t.SubType - 128)
	bytesRead := 0
	palette := make([]CPixel, paletteSize)

	for x, _ := range palette {
		pixel := make(CPixel, t.BytesPerCPixel)
		n, err := io.ReadAtLeast(buf, pixel, len(pixel))
		bytesRead += n
		if err != nil {
			return bytesRead, err
		}
		palette[x] = pixel
	}

	temp := 0
	runs := 0

	for temp < int(t.Width*t.Height) {
		runs++
		paletteIndexArr := make([]byte, 1)
		n, err := io.ReadAtLeast(buf, paletteIndexArr, 1)
		if err != nil {
			return bytesRead, err
		}
		bytesRead += n
		index := paletteIndexArr[0]

		var colour CPixel
		log.Println("    Subencoding:", t.SubType, "RawIndex:", index, "Index:", index&127, "Palette:", len(palette))
		if index < 128 {
			colour = palette[index]
			t.Pixels = append(t.Pixels, colour)
			temp++
		} else {
			colour = palette[index-128]
			runLength, _ := CalcRuns(buf, 255)
			for i := 0; i < runLength; i++ {
				t.Pixels = append(t.Pixels, colour)
				temp++
			}
		}
	}
	return bytesRead, nil
}

// CalcRuns calculates the length of a single-pixel run, in bytes
func CalcRuns(buffer io.Reader, maxVal int) (int, int) {
	bytesRead := 0
	length := 1
	p := make([]byte, 1)
	for {
		io.ReadAtLeast(buffer, p, 1)
		bytesRead++
		length += int(p[0])
		if int(p[0]) != maxVal {
			return length, bytesRead
		}
	}
}

// GetSubencoding returns the sub-encoding of a ZRLE data stream
func GetSubencoding(b byte) (encoding Subencoding, err error) {
	switch {
	case b == 0:
		encoding = RawEncoding{}
	case b == 1:
		encoding = SolidEncoding{}
	// TODO: Decide whether this should cut off at type 17 or type 128
	case b < 128:
		encoding = PackedPaletteEncoding{}
	case b == 128:
		encoding = RleEncoding{}
	case b >= 130:
		encoding = PrleEncoding{}
	default:
		err = fmt.Errorf("Invalid encoding type: %v", b)
	}
	return
}
