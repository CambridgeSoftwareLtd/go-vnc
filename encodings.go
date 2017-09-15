/*
Implementation of RFC 6143 §7.7 Encodings.
https://tools.ietf.org/html/rfc6143#section-7.7
*/
package vnc

import (
	"bytes"
	"fmt"

	"encoding/binary"
	"log"

	"io"

	"github.com/CambridgeSoftwareLtd/go-vnc/encodings"
	"github.com/CambridgeSoftwareLtd/go-vnc/rfbflags"
	"github.com/CambridgeSoftwareLtd/go-vnc/zrle"
)

//=============================================================================
// Encodings

// An Encoding implements a method for encoding pixel data that is
// sent by the server to the client.
type Encoding interface {
	fmt.Stringer
	Marshaler

	// Read the contents of the encoded pixel data from the reader.
	// This should return a new Encoding implementation that contains
	// the proper data.
	Read(*ClientConn, *Rectangle) (Encoding, error)

	// The number that uniquely identifies this encoding type.
	Type() encodings.Encoding
}

// Encodings describes a slice of Encoding.
type Encodings []Encoding

// Verify that interfaces are honored.
var _ Marshaler = (*Encodings)(nil)

// Marshal implements the Marshaler interface.
func (e Encodings) Marshal() ([]byte, error) {
	buf := NewBuffer(nil)
	for _, enc := range e {
		if err := buf.Write(enc.Type()); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

//-----------------------------------------------------------------------------
// Raw Encoding
//
// Raw encoding is the simplest encoding type, which is raw pixel data.
//
// See RFC 6143 §7.7.1.
// https://tools.ietf.org/html/rfc6143#section-7.7.1

// RawEncoding holds raw encoded rectangle data.
type RawEncoding struct {
	Colors []Color
}

// Verify that interfaces are honored.
var _ Encoding = (*RawEncoding)(nil)

// Marshal implements the Encoding interface.
func (e *RawEncoding) Marshal() ([]byte, error) {
	buf := NewBuffer(nil)

	for _, c := range e.Colors {
		bytes, err := c.Marshal()
		if err != nil {
			return nil, err
		}
		if err := buf.Write(bytes); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// Read implements the Encoding interface.
func (*RawEncoding) Read(c *ClientConn, rect *Rectangle) (Encoding, error) {
	var buf bytes.Buffer
	bytesPerPixel := int(c.pixelFormat.BPP / 8)

	colors := make([]Color, rect.Area())
	for y := uint16(0); y < rect.Height; y++ {
		for x := uint16(0); x < rect.Width; x++ {
			if err := c.receiveN(&buf, bytesPerPixel); err != nil {
				return nil, fmt.Errorf("unable to read rectangle with raw encoding: %s", err)
			}

			color := NewColor(&c.pixelFormat, &c.colorMap)
			if err := color.Unmarshal(buf.Next(bytesPerPixel)); err != nil {
				return nil, err
			}
			colors[int(y)*int(rect.Width)+int(x)] = *color
		}
	}

	return &RawEncoding{colors}, nil
}

// String implements the fmt.Stringer interface.
func (*RawEncoding) String() string { return "RawEncoding" }

// Type implements the Encoding interface.
func (*RawEncoding) Type() encodings.Encoding { return encodings.Raw }

//-----------------------------------------------------------------------------
// CopyRect Encoding
//
// CopyRect encoding defines a source pixel to copy data from
//
// See RFC 6143 §7.7.2.
// https://tools.ietf.org/html/rfc6143#section-7.7.2

type CopyRectEncoding struct {
	X uint16
	Y uint16
}

// Verify that interfaces are honored.
var _ Encoding = (*CopyRectEncoding)(nil)

// Marshal implements the Marshaler interface.
func (*CopyRectEncoding) Marshal() ([]byte, error) {
	return []byte{}, nil
}

// Read implements the Encoding interface.
func (*CopyRectEncoding) Read(c *ClientConn, rect *Rectangle) (Encoding, error) {
	var buf bytes.Buffer

	// X and Y have 2 bytes each
	if err := c.receiveN(&buf, 4); err != nil {
		return nil, fmt.Errorf("unable to read CopyRectEncoding buffer: %s", err)
	}

	x := binary.BigEndian.Uint16(buf.Next(2))
	y := binary.BigEndian.Uint16(buf.Next(2))

	return &CopyRectEncoding{X: x, Y: y}, nil
}

// String implements the fmt.Stringer interface.
func (*CopyRectEncoding) String() string { return "CopyRect" }

// Type implements the Encoding interface.
func (*CopyRectEncoding) Type() encodings.Encoding { return encodings.CopyRect }

//-----------------------------------------------------------------------------
// RRE Encoding
//
// RRE encoding defines a 2D rectangle of pixel data to draw
//
// See RFC 6143 §7.7.3.
// https://tools.ietf.org/html/rfc6143#section-7.7.3

// RREncoding represents an RRE encoded framebufferupdate
type RREncoding struct {
	NumSubRects uint32
	BackColour  Color
	Rects       []RRERect
}

type RRERect struct {
	BackColour Color
	X          uint16
	Y          uint16
	Width      uint16
	Height     uint16
}

// Verify that interfaces are honored.
var _ Encoding = (*RREncoding)(nil)

// Marshal implements the Marshaler interface.
func (e *RREncoding) Marshal() ([]byte, error) {
	return []byte{}, nil
}

// Read implements the Encoding interface.
func (*RREncoding) Read(c *ClientConn, rect *Rectangle) (Encoding, error) {
	var buf bytes.Buffer
	bytesPerPixel := int(c.pixelFormat.BPP / 8)

	// 4 bytes for nSubRects
	if err := c.receiveN(&buf, 4); err != nil {
		return nil, fmt.Errorf("unable to read RRE nSubRects: %s", err)
	}
	nSubRects := binary.BigEndian.Uint32(buf.Next(4))

	// bytesPerPixel bytes for background color
	if err := c.receiveN(&buf, bytesPerPixel); err != nil {
		return nil, fmt.Errorf("unable to read RRE backgroundColor: %s", err)
	}
	backPixVal := NewColor(&c.pixelFormat, &c.colorMap)
	if err := backPixVal.Unmarshal(buf.Next(bytesPerPixel)); err != nil {
		return nil, fmt.Errorf("unable to convert RRE backgroundColor: %s", err)
	}

	rects := make([]RRERect, nSubRects)

	for i := 0; i < int(nSubRects); i++ {
		n := (8 + bytesPerPixel)
		if err := c.receiveN(&buf, n); err != nil {
			return nil, fmt.Errorf("unable to read RRE subRects: %s", err)
		}

		subRectPixVal := NewColor(&c.pixelFormat, &c.colorMap)
		if err := subRectPixVal.Unmarshal(buf.Next(bytesPerPixel)); err != nil {
			return nil, fmt.Errorf("unable to read RRE sub-rectangle(%v) backgroundColor: %s", i, err)
		}
		rect := RRERect{
			BackColour: *subRectPixVal,
			X:          binary.BigEndian.Uint16(buf.Next(2)),
			Y:          binary.BigEndian.Uint16(buf.Next(2)),
			Width:      binary.BigEndian.Uint16(buf.Next(2)),
			Height:     binary.BigEndian.Uint16(buf.Next(2)),
		}

		rects = append(rects, rect)
	}

	// TODO: convert & return pixels
	return &RREncoding{nSubRects, *backPixVal, rects}, nil
}

// String implements the fmt.Stringer interface.
func (e *RREncoding) String() string { return "RREEncoding" }

// Type implements the Encoding interface.
func (*RREncoding) Type() encodings.Encoding { return encodings.RRE }

//-----------------------------------------------------------------------------
// ZRLE Encoding
//
// ZRLE encoding combines zlib compression, tiling, palettisation and run-length encoding
//
// See RFC 6143 §7.7.6.
// https://tools.ietf.org/html/rfc6143#section-7.7.6

// ZRLEncoding represents an ZRLE encoded update
type ZRLEncoding struct {
	Length     uint32
	ColourData [][]zrle.CPixel
}

// Verify that interfaces are honored.
var _ Encoding = (*ZRLEncoding)(nil)

// Marshal implements the Marshaler interface.
func (z *ZRLEncoding) Marshal() ([]byte, error) {
	return []byte{}, nil
}

// Read implements the Encoding interface.
func (z *ZRLEncoding) Read(c *ClientConn, rect *Rectangle) (Encoding, error) {
	log.Printf("  ZRLE - %v, %v, %v, %v", rect.X, rect.Y, rect.Width, rect.Height)
	var buf bytes.Buffer

	// First 4 bytes are length of zlib encoded data
	n := 4
	if err := c.receiveN(&buf, n); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(buf.Next(n))

	// Remaining [length] bytes are the zlib encoded data
	if err := c.receiveN(&buf, int(length)); err != nil {
		log.Printf("ZLIB body retrieve error: %s", err)
		return nil, err
	}

	c.zlibStream.Write(buf.Bytes())

	colourData, err := z.Decode(c, rect)

	if err != nil {
		return nil, err
	}

	return &ZRLEncoding{length, colourData}, nil
}

// Decode Decodes the data attached to the ZLRE message
func (z *ZRLEncoding) Decode(c *ClientConn, rect *Rectangle) ([][]zrle.CPixel, error) {
	tiles := zrle.CreateTiles(int(rect.Width), int(rect.Height))

	for i := range tiles {

		tiles[i].BytesPerCPixel = int(c.pixelFormat.BPP / 8)
		if c.pixelFormat.TrueColor == rfbflags.RFBFlag(0) && c.pixelFormat.BPP == 32 && c.pixelFormat.Depth <= 24 {
			tiles[i].BytesPerCPixel = 3
		}

		// TODO: remove this - testing against python version
		tiles[i].BytesPerCPixel = 3

		p := make([]byte, 1)
		_, err := io.ReadAtLeast(&c.zlibStream, p, 1)

		if err != nil {
			log.Printf("\tFirst byte parse error (tile %d): %s", i, err)
			return nil, err
		}

		s := p[0]

		se, err := zrle.GetSubencoding(s)
		if err != nil {
			return nil, err
		}

		tiles[i].SubType = int(s)
		_, err = se.Read(&c.zlibStream, &tiles[i])
		if err != nil {
			return nil, err
		}

	}

	drawData := zrle.TilesToPixels(int(rect.Width), int(rect.Height), tiles)

	return drawData, nil
}

// String implements the fmt.Stringer interface.
func (z *ZRLEncoding) String() string { return "ZRLEncoding" }

// Type implements the Encoding interface.
func (z *ZRLEncoding) Type() encodings.Encoding { return encodings.ZRLE }

//-----------------------------------------------------------------------------
// Cursor Pseudo-Encoding
//
// A client that requests the Cursor pseudo-encoding is declaring that
// it is capable of drawing a pointer cursor locally.  This can
// significantly improve perceived performance over slow links.  The
// server sets the cursor shape by sending a rectangle with the Cursor
//
// See RFC 6143 §7.8.1.
// https://tools.ietf.org/html/rfc6143#section-7.8.1

// CursorPseudoEncoding represents a Cursor message from the server.
type CursorPseudoEncoding struct {
	cursorPixels []Color
	bitmask      []uint8
}

// Verify that interfaces are honored.
var _ Encoding = (*CursorPseudoEncoding)(nil)

// Marshal implements the Marshaler interface.
func (e *CursorPseudoEncoding) Marshal() ([]byte, error) {
	return []byte{}, nil
}

// Read implements the Encoding interface.
func (*CursorPseudoEncoding) Read(c *ClientConn, rect *Rectangle) (Encoding, error) {
	var cursorPixels, bitmask bytes.Buffer
	bytesPerPixel := int(c.pixelFormat.BPP / 8)

	n := rect.Area() * bytesPerPixel
	if err := c.receiveN(&cursorPixels, n); err != nil {
		return nil, fmt.Errorf("unable to read cursorpixels: %s", err)
	}

	n = int(rect.Height) * ((int(rect.Width) + 7) / 8)
	if err := c.receiveN(&bitmask, n); err != nil {
		return nil, fmt.Errorf("unable to read bitmask: %s", err)
	}

	// TODO: convert & return cursorPixels / bitmask
	return &CursorPseudoEncoding{}, nil
}

// String implements the fmt.Stringer interface.
func (e *CursorPseudoEncoding) String() string { return "CursorPseudoEncoding" }

// Type implements the Encoding interface.
func (*CursorPseudoEncoding) Type() encodings.Encoding { return encodings.CursorPseudo }

//-----------------------------------------------------------------------------
// DesktopSize Pseudo-Encoding
//
// When a client requests DesktopSize pseudo-encoding, it is indicating to the
// server that it can handle changes to the framebuffer size. If this encoding
// received, the client must resize its framebuffer, and drop all existing
// information stored in the framebuffer.
//
// See RFC 6143 §7.8.2.
// https://tools.ietf.org/html/rfc6143#section-7.8.2

// DesktopSizePseudoEncoding represents a desktop size message from the server.
type DesktopSizePseudoEncoding struct{}

// Verify that interfaces are honored.
var _ Encoding = (*DesktopSizePseudoEncoding)(nil)

// Marshal implements the Marshaler interface.
func (e *DesktopSizePseudoEncoding) Marshal() ([]byte, error) {
	return []byte{}, nil
}

// Read implements the Encoding interface.
func (*DesktopSizePseudoEncoding) Read(c *ClientConn, rect *Rectangle) (Encoding, error) {
	c.fbWidth = rect.Width
	c.fbHeight = rect.Height

	return &DesktopSizePseudoEncoding{}, nil
}

// String implements the fmt.Stringer interface.
func (e *DesktopSizePseudoEncoding) String() string { return "DesktopSizePseudoEncoding" }

// Type implements the Encoding interface.
func (*DesktopSizePseudoEncoding) Type() encodings.Encoding { return encodings.DesktopSizePseudo }
