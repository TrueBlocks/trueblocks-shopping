// PDF Export Utility Functions

import type { ColorInfo, ShoppingListItem, PdfPaletteItem, ProcessingResult } from './types';

// Color distance threshold for highlighting similar pixels
const COLOR_THRESHOLD = 30;

// Calculate color distance (Euclidean)
export function colorDistance(r1: number, g1: number, b1: number, r2: number, g2: number, b2: number): number {
  return Math.sqrt(
    Math.pow(r1 - r2, 2) +
    Math.pow(g1 - g2, 2) +
    Math.pow(b1 - b2, 2)
  );
}

// Get image from canvas as base64
export function canvasToBase64(canvas: HTMLCanvasElement | null): string | null {
  if (!canvas) return null;
  return canvas.toDataURL('image/png');
}

// Convert ImageData to base64
export function imageDataToBase64(imageData: ImageData | null): string | null {
  if (!imageData) return null;
  const canvas = document.createElement('canvas');
  canvas.width = imageData.width;
  canvas.height = imageData.height;
  const ctx = canvas.getContext('2d');
  if (!ctx) return null;
  ctx.putImageData(imageData, 0, 0);
  return canvas.toDataURL('image/png');
}

// Apply smoothing passes to image data
export function applySmoothing(imageData: ImageData, passes: number): ImageData {
  if (passes <= 0) return imageData;
  
  const width = imageData.width;
  const height = imageData.height;
  let currentData = new Uint8ClampedArray(imageData.data);
  
  for (let pass = 0; pass < passes; pass++) {
    const newData = new Uint8ClampedArray(currentData);
    
    for (let y = 1; y < height - 1; y++) {
      for (let x = 1; x < width - 1; x++) {
        const idx = (y * width + x) * 4;
        
        // Count colors in 3x3 neighborhood
        const colorCounts = new Map<string, { count: number; r: number; g: number; b: number }>();
        
        for (let dy = -1; dy <= 1; dy++) {
          for (let dx = -1; dx <= 1; dx++) {
            const nIdx = ((y + dy) * width + (x + dx)) * 4;
            const key = `${currentData[nIdx]},${currentData[nIdx + 1]},${currentData[nIdx + 2]}`;
            const existing = colorCounts.get(key);
            if (existing) {
              existing.count++;
            } else {
              colorCounts.set(key, {
                count: 1,
                r: currentData[nIdx],
                g: currentData[nIdx + 1],
                b: currentData[nIdx + 2],
              });
            }
          }
        }
        
        // Find most common color
        let maxCount = 0;
        let dominantColor = { r: currentData[idx], g: currentData[idx + 1], b: currentData[idx + 2] };
        colorCounts.forEach((value) => {
          if (value.count > maxCount) {
            maxCount = value.count;
            dominantColor = { r: value.r, g: value.g, b: value.b };
          }
        });
        
        newData[idx] = dominantColor.r;
        newData[idx + 1] = dominantColor.g;
        newData[idx + 2] = dominantColor.b;
      }
    }
    
    currentData = newData;
  }
  
  return new ImageData(currentData, width, height);
}

// Apply downsampling (tile effect)
export function applyDownsampling(imageData: ImageData, tileSize: number): ImageData {
  if (tileSize <= 1) return imageData;
  
  const width = imageData.width;
  const height = imageData.height;
  const data = imageData.data;
  const newData = new Uint8ClampedArray(data);
  
  for (let ty = 0; ty < height; ty += tileSize) {
    for (let tx = 0; tx < width; tx += tileSize) {
      // Calculate average color for tile
      let sumR = 0, sumG = 0, sumB = 0, count = 0;
      
      for (let dy = 0; dy < tileSize && ty + dy < height; dy++) {
        for (let dx = 0; dx < tileSize && tx + dx < width; dx++) {
          const idx = ((ty + dy) * width + (tx + dx)) * 4;
          sumR += data[idx];
          sumG += data[idx + 1];
          sumB += data[idx + 2];
          count++;
        }
      }
      
      const avgR = Math.round(sumR / count);
      const avgG = Math.round(sumG / count);
      const avgB = Math.round(sumB / count);
      
      // Apply average to all pixels in tile
      for (let dy = 0; dy < tileSize && ty + dy < height; dy++) {
        for (let dx = 0; dx < tileSize && tx + dx < width; dx++) {
          const idx = ((ty + dy) * width + (tx + dx)) * 4;
          newData[idx] = avgR;
          newData[idx + 1] = avgG;
          newData[idx + 2] = avgB;
        }
      }
    }
  }
  
  return new ImageData(newData, width, height);
}

// Get processed source image data (considering posterize, smoothing, tiling)
export function getProcessedSourceImageData(
  originalImageData: ImageData,
  posterizedImageData: ImageData | null,
  posterizeMode: boolean,
  smoothingPasses: number,
  tileSize: number
): ImageData {
  const usePosterized = posterizeMode && posterizedImageData;
  if (!usePosterized) return originalImageData;
  
  let processed = posterizedImageData;
  if (smoothingPasses > 0) {
    processed = applySmoothing(processed, smoothingPasses);
  }
  if (tileSize > 1) {
    processed = applyDownsampling(processed, tileSize);
  }
  return processed;
}

// Collect unique paints from all mixing recipes into a shopping list
export function buildShoppingList(result: ProcessingResult): ShoppingListItem[] {
  const paintMap = new Map<string, ShoppingListItem>();
  
  result.palette.forEach(item => {
    item.mixingRecipe?.forEach(recipe => {
      const key = recipe.paint.id || recipe.paint.name;
      if (!paintMap.has(key)) {
        paintMap.set(key, {
          name: recipe.paint.name,
          brand: recipe.paint.brand,
          series: recipe.paint.series,
          pigments: recipe.paint.pigments,
          opacity: recipe.paint.opacity,
          hex: recipe.paint.hex,
        });
      }
    });
  });
  
  return Array.from(paintMap.values());
}

// Generate isolation image for a specific color (white background, only matching pixels shown)
export function generateIsolationImage(
  color: ColorInfo,
  sourceImageData: ImageData,
  usePosterized: boolean
): string {
  const tempCanvas = document.createElement('canvas');
  tempCanvas.width = sourceImageData.width;
  tempCanvas.height = sourceImageData.height;
  const tempCtx = tempCanvas.getContext('2d');
  if (!tempCtx) return '';
  
  tempCtx.putImageData(sourceImageData, 0, 0);
  const imageData = tempCtx.getImageData(0, 0, tempCanvas.width, tempCanvas.height);
  const data = imageData.data;
  
  for (let i = 0; i < data.length; i += 4) {
    const r = data[i];
    const g = data[i + 1];
    const b = data[i + 2];
    
    // For posterized images, use exact color match; for original, use threshold
    const isMatch = usePosterized
      ? (r === color.r && g === color.g && b === color.b)
      : (colorDistance(r, g, b, color.r, color.g, color.b) < COLOR_THRESHOLD);
    
    if (isMatch) {
      data[i] = color.r;
      data[i + 1] = color.g;
      data[i + 2] = color.b;
    } else {
      data[i] = 255;
      data[i + 1] = 255;
      data[i + 2] = 255;
    }
  }
  
  tempCtx.putImageData(imageData, 0, 0);
  return tempCanvas.toDataURL('image/png');
}

// Build palette data for PDF export (with isolation images)
export function buildPaletteForPdf(
  result: ProcessingResult,
  sourceImageData: ImageData,
  usePosterized: boolean
): PdfPaletteItem[] {
  return result.palette.map((item, index) => ({
    hex: item.dominantColor.hex,
    colorNumber: index + 1,
    mixingRecipe: item.mixingRecipe?.map(recipe => ({
      name: recipe.paint.name,
      parts: recipe.parts,
      hex: recipe.paint.hex,
    })) || [],
    isolationImageData: generateIsolationImage(item.dominantColor, sourceImageData, usePosterized),
  }));
}

// ============================================
// Paint-by-Numbers Image Generation
// ============================================

// Simple region info (just center and color)
interface SimpleRegion {
  colorIndex: number;
  centerX: number;
  centerY: number;
  pixelCount: number;
}

// Find which palette color a pixel belongs to
function findColorIndex(
  r: number, g: number, b: number,
  palette: ColorInfo[],
  usePosterized: boolean
): number {
  if (usePosterized) {
    // Exact match for posterized images
    for (let i = 0; i < palette.length; i++) {
      const c = palette[i];
      if (r === c.r && g === c.g && b === c.b) {
        return i;
      }
    }
  }
  
  // Find closest color
  let minDist = Infinity;
  let closestIdx = 0;
  for (let i = 0; i < palette.length; i++) {
    const c = palette[i];
    const dist = colorDistance(r, g, b, c.r, c.g, c.b);
    if (dist < minDist) {
      minDist = dist;
      closestIdx = i;
    }
  }
  return closestIdx;
}

// Create a color index map for the image
function createColorIndexMap(
  imageData: ImageData,
  palette: ColorInfo[],
  usePosterized: boolean
): Int32Array {
  const { width, height, data } = imageData;
  const indexMap = new Int32Array(width * height);
  
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const pixelIdx = y * width + x;
      const dataIdx = pixelIdx * 4;
      const r = data[dataIdx];
      const g = data[dataIdx + 1];
      const b = data[dataIdx + 2];
      indexMap[pixelIdx] = findColorIndex(r, g, b, palette, usePosterized);
    }
  }
  
  return indexMap;
}

// Apply simple mode filter (most common color in neighborhood)
function applyModeFilterFast(indexMap: Int32Array, width: number, height: number, radius: number): Int32Array {
  const result = new Int32Array(indexMap.length);
  
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const counts: number[] = [];
      
      // Sample neighborhood
      for (let dy = -radius; dy <= radius; dy++) {
        for (let dx = -radius; dx <= radius; dx++) {
          const nx = x + dx;
          const ny = y + dy;
          if (nx >= 0 && nx < width && ny >= 0 && ny < height) {
            const color = indexMap[ny * width + nx];
            counts[color] = (counts[color] || 0) + 1;
          }
        }
      }
      
      // Find most common color
      let maxCount = 0;
      let modeColor = indexMap[y * width + x];
      for (let c = 0; c < counts.length; c++) {
        if (counts[c] > maxCount) {
          maxCount = counts[c];
          modeColor = c;
        }
      }
      
      result[y * width + x] = modeColor;
    }
  }
  
  return result;
}

// Find contiguous regions using flood fill with typed arrays
function findRegionsFast(
  indexMap: Int32Array,
  width: number,
  height: number
): { regionMap: Int32Array; regions: SimpleRegion[] } {
  const regionMap = new Int32Array(width * height).fill(-1);
  const regions: SimpleRegion[] = [];
  let regionId = 0;
  
  for (let startY = 0; startY < height; startY++) {
    for (let startX = 0; startX < width; startX++) {
      const startIdx = startY * width + startX;
      if (regionMap[startIdx] !== -1) continue;
      
      const colorIndex = indexMap[startIdx];
      let sumX = 0, sumY = 0, count = 0;
      
      // Simple scanline flood fill
      const stack: number[] = [startIdx];
      
      while (stack.length > 0) {
        const idx = stack.pop()!;
        if (regionMap[idx] !== -1) continue;
        if (indexMap[idx] !== colorIndex) continue;
        
        regionMap[idx] = regionId;
        const x = idx % width;
        const y = Math.floor(idx / width);
        sumX += x;
        sumY += y;
        count++;
        
        // Add neighbors (check bounds)
        if (x > 0) stack.push(idx - 1);
        if (x < width - 1) stack.push(idx + 1);
        if (y > 0) stack.push(idx - width);
        if (y < height - 1) stack.push(idx + width);
      }
      
      if (count > 0) {
        regions.push({
          colorIndex,
          centerX: Math.round(sumX / count),
          centerY: Math.round(sumY / count),
          pixelCount: count,
        });
        regionId++;
      }
    }
  }
  
  return { regionMap, regions };
}

/**
 * Generate a paint-by-numbers outline image
 * - White background
 * - Light grey outlines between color regions
 * - Numbers centered in each region
 * - Optional 4x4 grid overlay
 * - Optional highlighting of a specific color index
 */
export function generatePaintByNumbersImage(
  sourceImageData: ImageData,
  palette: ColorInfo[],
  usePosterized: boolean,
  showGrid: boolean = true,
  highlightColorIndex: number = -1 // -1 means no highlight
): string {
  const { width, height } = sourceImageData;
  
  // Create canvas
  const canvas = document.createElement('canvas');
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext('2d');
  if (!ctx) return '';
  
  // Fill with white background
  ctx.fillStyle = 'white';
  ctx.fillRect(0, 0, width, height);
  
  // Create color index map
  let indexMap = createColorIndexMap(sourceImageData, palette, usePosterized);
  
  // Apply aggressive mode filtering to merge small regions
  // Multiple passes with increasing radius for maximum simplification
  indexMap = applyModeFilterFast(indexMap, width, height, 3);
  indexMap = applyModeFilterFast(indexMap, width, height, 5);
  indexMap = applyModeFilterFast(indexMap, width, height, 7);
  indexMap = applyModeFilterFast(indexMap, width, height, 5);
  
  // Find contiguous regions
  const { regionMap, regions } = findRegionsFast(indexMap, width, height);
  
  // Draw light grey outlines where region changes
  ctx.fillStyle = '#BBBBBB';
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const idx = y * width + x;
      const currentRegion = regionMap[idx];
      
      // Check if this is an edge pixel
      let isEdge = false;
      if (x === 0 || x === width - 1 || y === 0 || y === height - 1) {
        isEdge = true;
      } else {
        // Check 4 neighbors
        if (regionMap[idx - 1] !== currentRegion ||
            regionMap[idx + 1] !== currentRegion ||
            regionMap[idx - width] !== currentRegion ||
            regionMap[idx + width] !== currentRegion) {
          isEdge = true;
        }
      }
      
      if (isEdge) {
        ctx.fillRect(x, y, 1, 1);
      }
    }
  }
  
  // Calculate font sizes based on image dimensions
  const baseFontSize = Math.max(12, Math.min(20, Math.floor(Math.min(width, height) / 30)));
  const highlightFontSize = Math.floor(baseFontSize * 1.5);
  
  // Very low minimum region size - show numbers on almost all regions
  const minSizeForLabel = baseFontSize * baseFontSize;
  
  // Draw numbers in region centers
  for (const region of regions) {
    if (region.pixelCount < minSizeForLabel) continue;
    
    const label = (region.colorIndex + 1).toString();
    const isHighlighted = region.colorIndex === highlightColorIndex;
    
    if (isHighlighted) {
      // Highlighted: larger, bold, darker
      ctx.font = `bold ${highlightFontSize}px Arial`;
      ctx.fillStyle = '#000000';
    } else {
      // Normal: regular size, grey
      ctx.font = `${baseFontSize}px Arial`;
      ctx.fillStyle = '#666666';
    }
    
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(label, region.centerX, region.centerY);
  }
  
  // Draw 4x4 grid overlay
  if (showGrid) {
    ctx.strokeStyle = '#999999';
    ctx.lineWidth = 1;
    ctx.setLineDash([8, 4]);
    
    // 3 vertical lines
    for (let i = 1; i <= 3; i++) {
      const x = Math.round((width * i) / 4);
      ctx.beginPath();
      ctx.moveTo(x, 0);
      ctx.lineTo(x, height);
      ctx.stroke();
    }
    
    // 3 horizontal lines
    for (let i = 1; i <= 3; i++) {
      const y = Math.round((height * i) / 4);
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(width, y);
      ctx.stroke();
    }
    
    ctx.setLineDash([]);
  }
  
  return canvas.toDataURL('image/png');
}

/**
 * Generate a minimal paint-by-numbers image for full-page display
 * - White background
 * - Very light, thin dashed outlines
 * - Small but readable numbers
 * - No grid overlay
 */
export function generatePaintByNumbersImageMinimal(
  sourceImageData: ImageData,
  palette: ColorInfo[],
  usePosterized: boolean
): string {
  const { width, height } = sourceImageData;
  
  // Create canvas
  const canvas = document.createElement('canvas');
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext('2d');
  if (!ctx) return '';
  
  // Fill with white background
  ctx.fillStyle = 'white';
  ctx.fillRect(0, 0, width, height);
  
  // Create color index map
  let indexMap = createColorIndexMap(sourceImageData, palette, usePosterized);
  
  // Apply aggressive mode filtering to merge small regions
  indexMap = applyModeFilterFast(indexMap, width, height, 3);
  indexMap = applyModeFilterFast(indexMap, width, height, 5);
  indexMap = applyModeFilterFast(indexMap, width, height, 7);
  indexMap = applyModeFilterFast(indexMap, width, height, 5);
  
  // Find contiguous regions
  const { regionMap, regions } = findRegionsFast(indexMap, width, height);
  
  // Collect edge pixels for dashed outline
  const edgePixels: Array<{x: number, y: number}> = [];
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const idx = y * width + x;
      const currentRegion = regionMap[idx];
      
      let isEdge = false;
      if (x === 0 || x === width - 1 || y === 0 || y === height - 1) {
        isEdge = true;
      } else {
        if (regionMap[idx - 1] !== currentRegion ||
            regionMap[idx + 1] !== currentRegion ||
            regionMap[idx - width] !== currentRegion ||
            regionMap[idx + width] !== currentRegion) {
          isEdge = true;
        }
      }
      
      if (isEdge) {
        edgePixels.push({x, y});
      }
    }
  }
  
  // Draw very light dashed outlines (only every 3rd pixel for dashed effect)
  ctx.fillStyle = '#DDDDDD';
  for (let i = 0; i < edgePixels.length; i++) {
    // Skip 2 out of every 3 pixels for dashed effect
    if (i % 3 === 0) {
      const {x, y} = edgePixels[i];
      ctx.fillRect(x, y, 1, 1);
    }
  }
  
  // Small but readable font size
  const fontSize = Math.max(8, Math.min(14, Math.floor(Math.min(width, height) / 50)));
  
  // Show numbers on regions large enough
  const minSizeForLabel = fontSize * fontSize;
  
  // Draw numbers in region centers
  ctx.font = `${fontSize}px Arial`;
  ctx.fillStyle = '#888888';
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  
  for (const region of regions) {
    if (region.pixelCount < minSizeForLabel) continue;
    const label = (region.colorIndex + 1).toString();
    ctx.fillText(label, region.centerX, region.centerY);
  }
  
  return canvas.toDataURL('image/png');
}

