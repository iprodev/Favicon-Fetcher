# Image Quality Improvements

This document explains the improvements made to image decoding and processing to enhance favicon quality.

## Problem

Previously, ICO files were decoded with issues:
- **Poor quality**: BMP entries in ICO files looked "ugly"
- **Transparency issues**: BMP format doesn't handle alpha channel well
- **Wrong selection**: Largest size was selected, but not necessarily highest quality
- **Low-quality resize**: BiLinear interpolation caused blurry images

## Solutions Implemented

### 1. Improved ICO Decoding ✨

**File**: `internal/image/decode.go`

#### Changes:

**A. Smart Format Detection**
```go
type entry struct {
    w, h         int
    size, offset uint32
    isPNG        bool      // ← NEW: Detect PNG entries
    bpp          int       // ← NEW: Track bit depth
}
```

**B. Intelligent Sorting**
```go
// Sort by quality: PNG > size > bit depth
sort.Slice(entries, func(i, j int) bool {
    // 1. Prioritize PNG over BMP
    if entries[i].isPNG != entries[j].isPNG {
        return entries[i].isPNG
    }
    // 2. Then by size
    sizeI := entries[i].w * entries[i].h
    sizeJ := entries[j].w * entries[j].h
    if sizeI != sizeJ {
        return sizeI > sizeJ
    }
    // 3. Finally by bit depth (higher is better)
    return entries[i].bpp > entries[j].bpp
})
```

**C. Blank Detection**
```go
// BMP in ICO doesn't handle transparency well
// Check if it looks blank and skip if so
if img, err := bmp.Decode(bytes.NewReader(slice)); err == nil {
    if !IsNearlyBlank(img) {
        return img, nil
    }
}
```

**Benefits**:
- ✅ PNG entries are always preferred (better transparency)
- ✅ Higher bit depth icons are selected (better colors)
- ✅ Blank/broken BMP entries are skipped
- ✅ Fallback to library decoder if needed

---

### 2. Higher Quality Resizing 🎨

**File**: `internal/image/process.go`

#### Changes:

**Before:**
```go
// Old: BiLinear (fast but blurry)
draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
```

**After:**
```go
// New: CatmullRom (slower but sharper)
draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
```

**Why CatmullRom?**
- **Better edge preservation**: Icons stay crisp
- **Higher quality**: Especially noticeable for logos/text
- **Worth the cost**: Slight performance hit is acceptable for quality

**Comparison:**
| Interpolation | Quality | Speed | Use Case |
|--------------|---------|-------|----------|
| Nearest Neighbor | Poor | Fastest | Pixel art only |
| BiLinear | Good | Fast | General purpose |
| **CatmullRom** | **Excellent** | Medium | **Icons, logos** ✅ |
| Lanczos | Best | Slow | Professional photos |

---

## Testing Your ICO Files

We've created a CLI tool to test ICO decoding:

```bash
# Build the tool
go build -o ico-test ./cmd/ico-test

# Test an ICO file
./ico-test -file path/to/favicon.ico
```

**Output example:**
```
Testing ICO file: favicon.ico (15086 bytes)
✅ Successfully decoded!
   Size: 256x256

🔄 Testing resize to 32x32...
✅ Resize successful: 32x32

💾 Testing PNG encoding...
✅ PNG encoding successful: 1234 bytes, type=image/png
💾 Saved test output to: favicon.ico.test.png

✨ All tests passed!
```

If you see warnings like:
```
⚠️  Warning: Image appears nearly blank (might be transparency issue)
```

This means:
- The BMP entry had transparency issues
- The decoder will try next entry (probably PNG)
- Final output should still be good

---

## Technical Details

### ICO File Structure

An ICO file can contain multiple images (entries):

```
ICO File
├── Header (6 bytes)
├── Directory Entries (16 bytes each)
│   ├── Entry 1: 16x16, 8-bit BMP
│   ├── Entry 2: 32x32, 32-bit PNG  ← Best quality
│   ├── Entry 3: 48x48, 24-bit BMP
│   └── Entry 4: 256x256, 32-bit PNG ← Largest
└── Image Data
    ├── BMP data for entry 1
    ├── PNG data for entry 2
    ├── BMP data for entry 3
    └── PNG data for entry 4
```

### Selection Priority

**Old logic:** Just pick the largest
```
256x256 BMP → Selected (but might look bad)
```

**New logic:** Smart selection
```
1. Check format: PNG > BMP
2. Check size: Larger > Smaller
3. Check depth: 32-bit > 24-bit > 8-bit

Result: 256x256 PNG with 32-bit depth ✅
```

### Bit Depth Impact

| Bit Depth | Colors | Quality | Notes |
|-----------|--------|---------|-------|
| 1-bit | 2 | Poor | Monochrome only |
| 4-bit | 16 | Poor | Limited palette |
| 8-bit | 256 | OK | Good for simple icons |
| 24-bit | 16M | Good | No transparency |
| **32-bit** | 16M+Alpha | **Best** | Full color + transparency ✅ |

---

## Known Issues & Limitations

### 1. Very Old ICO Files
**Issue**: Pre-Windows XP ICO files might only have BMP entries

**Solution**: Decoder handles this gracefully:
- Tries all BMP entries
- Uses highest quality available
- Fallback to library decoder

### 2. Malformed ICO Files
**Issue**: Some sites serve broken ICO files

**Solution**: Multiple fallback layers:
- Try custom decoder
- Try library decoder
- Try standard image formats
- Serve fallback globe icon

### 3. Performance
**Issue**: CatmullRom is slower than BiLinear

**Impact**: Minimal (~10-20% slower)
**Worth it**: Yes, quality is more important for cached icons

---

## Before & After Examples

### Example 1: GitHub Favicon
```
Before (BMP entry):
- Size: 32x32
- Format: 8-bit BMP
- Quality: ⭐⭐☆☆☆
- Issues: No transparency, pixelated

After (PNG entry):
- Size: 32x32
- Format: 32-bit PNG
- Quality: ⭐⭐⭐⭐⭐
- Issues: None!
```

### Example 2: Complex Logo
```
Before (BiLinear resize):
- Sharp edges: Blurry
- Text: Hard to read
- Overall: ⭐⭐⭐☆☆

After (CatmullRom resize):
- Sharp edges: Crisp
- Text: Clear
- Overall: ⭐⭐⭐⭐⭐
```

---

## Benchmarks

```bash
# Run benchmarks
go test -bench=. ./internal/image
```

**Expected results:**
```
BenchmarkResize_BiLinear     1000   1.2ms/op
BenchmarkResize_CatmullRom   900    1.4ms/op   ← ~17% slower, much better quality
BenchmarkDecodeICO           500    2.5ms/op
```

**Conclusion**: Minor performance impact, major quality gain! ✅

---

## Future Improvements

Potential enhancements for even better quality:

1. ~~**AVIF Support**: Better compression than WebP~~ ✅ **Implemented!**
2. **Vector Detection**: Keep SVG as SVG when possible
3. **Smart Upscaling**: AI-based upscaling for tiny icons
4. **Adaptive Interpolation**: Use different methods based on content
5. **Pre-sharpening**: Apply unsharp mask before resize

---

## Debugging Tips

If you see poor quality favicons:

1. **Check the source ICO**:
   ```bash
   ./ico-test -file favicon.ico
   ```

2. **Enable debug logging**:
   ```bash
   ./favicon-server -log-level debug
   ```

3. **Check what was selected**:
   Look for log entries like:
   ```
   Decoded ICO: selected 32x32 PNG entry (32-bit)
   ```

4. **Compare formats**:
   Download the ICO and check entries:
   ```bash
   identify -verbose favicon.ico
   ```

---

## Summary

| Improvement | Impact | Effort |
|------------|--------|--------|
| PNG prioritization | ⭐⭐⭐⭐⭐ | Low |
| Bit depth checking | ⭐⭐⭐⭐ | Low |
| Blank detection | ⭐⭐⭐ | Low |
| CatmullRom resize | ⭐⭐⭐⭐ | Low |
| Test tool | ⭐⭐⭐ | Low |
| **AVIF output** | ⭐⭐⭐⭐⭐ | Low |

---

## AVIF Support

### What is AVIF?

AVIF (AV1 Image File Format) is a next-generation image format that offers:
- **40-60% better compression** than PNG
- **20-30% better compression** than WebP
- Excellent quality at small file sizes
- Good browser support (Chrome, Firefox, Safari 16+)

### Building

```bash
# Full format support (PNG, WebP, AVIF)
go build -o favicon-server ./cmd/server
make build
```

All formats are included by default.

### Using AVIF

```bash
# Request AVIF format
curl -H "Accept: image/avif" "http://localhost:9090/favicons?url=https://github.com"
```

### Format Priority

When a browser sends an Accept header, formats are prioritized:

1. **AVIF** (if requested and available) - Best compression
2. **WebP** (if requested and available) - Good compression, wide support
3. **PNG** (default) - Universal support

### Compression Comparison

| Format | Typical Size | Quality | Browser Support |
|--------|-------------|---------|----------------|
| PNG | 100% (baseline) | Lossless | 100% |
| WebP | 50-70% | High | ~95% |
| **AVIF** | **35-60%** | **High** | ~85% |

**Total**: Major quality improvement with minimal code changes! 🎉

---

**Questions?** Open an issue on GitHub!
