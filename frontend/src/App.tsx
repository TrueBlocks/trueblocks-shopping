import { useState, useEffect, useRef, useCallback } from 'react';
import './App.css';
import { EventsOn, LogInfo } from '../wailsjs/runtime/runtime';
import { GetNColors, SetNColors, GetTileSize, SetTileSize, GetPosterizeMode, SetPosterizeMode, GetSmoothingPasses, SetSmoothingPasses, GetAspectRatio, SetAspectRatio, ReprocessImage, ExportColorDetailPDF } from '../wailsjs/go/main/App';
import { main } from '../wailsjs/go/models';
import { exportComparisonPdf, exportPdf2, exportPdf3, exportPdf4, exportPdf5, type PdfExportContext } from './pdf';

// Types
type AspectRatioType = 'original' | 'landscape' | 'portrait' | 'square';

interface ColorInfo {
  r: number;
  g: number;
  b: number;
  hex: string;
}

interface PaintProduct {
  id: string;
  brand: string;
  name: string;
  series: number;
  opacity: string;
  pigments: string;
  rgb: number[];
  hex: string;
}

interface PaintMatch {
  paint: PaintProduct;
  deltaE: number;
  matchRating: string;
}

interface MixingPart {
  paint: PaintProduct;
  parts: number;
}

interface PaletteResult {
  dominantColor: ColorInfo;
  paintMatches: PaintMatch[];
  mixingRecipe: MixingPart[];
}

interface ProcessingResult {
  imageData: string;
  palette: PaletteResult[];
  imageWidth: number;
  imageHeight: number;
}

// Color distance threshold for highlighting similar pixels
const COLOR_THRESHOLD = 30;

function App() {
  const [isProcessing, setIsProcessing] = useState(false);
  const [progressMessage, setProgressMessage] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<ProcessingResult | null>(null);
  const [highlightedColorIndex, setHighlightedColorIndex] = useState<number | null>(null);
  const [lastColorIndex, setLastColorIndex] = useState<number>(0);
  const [originalImageData, setOriginalImageData] = useState<ImageData | null>(null);
  const [posterizedImageData, setPosterizedImageData] = useState<ImageData | null>(null);
  const [posterizeMode, setPosterizeModeState] = useState(false);
  const [tileSize, setTileSizeState] = useState(1);
  const [smoothingPasses, setSmoothingPassesState] = useState(0);
  const [aspectRatio, setAspectRatioState] = useState<AspectRatioType>('original');
  const [modalOpen, setModalOpen] = useState(false);
  const [nColors, setNColorsState] = useState(10);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const modalCanvasRef = useRef<HTMLCanvasElement>(null);
  const imageRef = useRef<HTMLImageElement | null>(null);
  const reprocessTimeoutRef = useRef<number | null>(null);

  // Load initial settings from backend
  useEffect(() => {
    GetNColors().then(setNColorsState);
    GetTileSize().then(setTileSizeState);
    GetPosterizeMode().then(setPosterizeModeState);
    GetSmoothingPasses().then(setSmoothingPassesState);
    GetAspectRatio().then((r) => setAspectRatioState(r as AspectRatioType));
  }, []);

  // Wrapper to sync state with backend
  const setNColors = useCallback((value: number) => {
    setNColorsState(value);
  }, []);

  const setTileSize = useCallback((value: number) => {
    setTileSizeState(value);
    SetTileSize(value);
  }, []);

  const setPosterizeMode = useCallback((value: boolean) => {
    setPosterizeModeState(value);
    SetPosterizeMode(value);
  }, []);

  const setSmoothingPasses = useCallback((value: number) => {
    setSmoothingPassesState(value);
    SetSmoothingPasses(value);
  }, []);

  const setAspectRatio = useCallback((value: AspectRatioType) => {
    setAspectRatioState(value);
    SetAspectRatio(value);
  }, []);

  // Handle slider change - update UI immediately, debounce reprocessing
  const handleNColorsChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const value = parseInt(e.target.value, 10);
    setNColors(value);
    SetNColors(value);
    
    // Clear any pending reprocess
    if (reprocessTimeoutRef.current) {
      clearTimeout(reprocessTimeoutRef.current);
    }
    
    // Debounce: only reprocess after 300ms of no slider movement
    if (result) {
      reprocessTimeoutRef.current = window.setTimeout(() => {
        ReprocessImage();
        reprocessTimeoutRef.current = null;
      }, 300);
    }
  }, [result]);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (reprocessTimeoutRef.current) {
        clearTimeout(reprocessTimeoutRef.current);
      }
    };
  }, []);

  // Register event listeners
  useEffect(() => {
    EventsOn('processing:started', () => {
      setIsProcessing(true);
      setError(null);
      setProgressMessage('Starting...');
      setHighlightedColorIndex(null);
      setOriginalImageData(null);
      setPosterizedImageData(null);
      // Note: Do NOT reset posterizeMode or tileSize - preserve user's view preference
    });

    EventsOn('processing:progress', (message: string) => {
      setProgressMessage(message);
    });

    EventsOn('processing:complete', (data: ProcessingResult) => {
      setResult(data);
      setIsProcessing(false);
      setProgressMessage('');
    });

    EventsOn('processing:error', (errorMsg: string) => {
      setError(errorMsg);
      setIsProcessing(false);
      setProgressMessage('');
    });
  }, []);

  // Apply smoothing to remove salt-and-pepper noise (paint-by-numbers mode)
  // Uses a mode filter: each pixel becomes the most common color in its neighborhood
  const applySmoothingPass = useCallback((imageData: ImageData, radius: number = 3): ImageData => {
    if (!result) return imageData;
    
    const palette = result.palette.map(p => p.dominantColor);
    const width = imageData.width;
    const height = imageData.height;
    const srcData = imageData.data;
    const output = new ImageData(
      new Uint8ClampedArray(srcData),
      width,
      height
    );
    const data = output.data;

    // For each pixel, find the most common palette color in its neighborhood
    for (let y = 0; y < height; y++) {
      for (let x = 0; x < width; x++) {
        const colorCounts = new Map<number, number>();
        
        // Sample neighborhood with distance weighting (closer pixels count more)
        for (let dy = -radius; dy <= radius; dy++) {
          for (let dx = -radius; dx <= radius; dx++) {
            const dist = Math.sqrt(dx * dx + dy * dy);
            if (dist > radius) continue; // Circular neighborhood
            
            const nx = Math.max(0, Math.min(width - 1, x + dx));
            const ny = Math.max(0, Math.min(height - 1, y + dy));
            const idx = (ny * width + nx) * 4;
            const r = srcData[idx];
            const g = srcData[idx + 1];
            const b = srcData[idx + 2];
            
            // Weight by distance (closer = more weight)
            const weight = Math.max(1, Math.round((radius - dist + 1)));
            
            // Find which palette index this color is
            for (let pi = 0; pi < palette.length; pi++) {
              const pc = palette[pi];
              if (pc.r === r && pc.g === g && pc.b === b) {
                colorCounts.set(pi, (colorCounts.get(pi) || 0) + weight);
                break;
              }
            }
          }
        }
        
        // Find most common color (lowest index wins ties)
        let maxCount = 0;
        let winningIndex = 0;
        for (const [colorIndex, count] of colorCounts.entries()) {
          if (count > maxCount || (count === maxCount && colorIndex < winningIndex)) {
            maxCount = count;
            winningIndex = colorIndex;
          }
        }
        
        const winningColor = palette[winningIndex] || palette[0];
        const idx = (y * width + x) * 4;
        data[idx] = winningColor.r;
        data[idx + 1] = winningColor.g;
        data[idx + 2] = winningColor.b;
      }
    }
    
    return output;
  }, [result]);

  // Morphological dilation: expand each color region
  const applyDilation = useCallback((imageData: ImageData, radius: number = 2): ImageData => {
    if (!result) return imageData;
    
    const palette = result.palette.map(p => p.dominantColor);
    const width = imageData.width;
    const height = imageData.height;
    const srcData = imageData.data;
    const output = new ImageData(new Uint8ClampedArray(srcData), width, height);
    const data = output.data;

    for (let y = 0; y < height; y++) {
      for (let x = 0; x < width; x++) {
        const colorCounts = new Map<number, number>();
        
        for (let dy = -radius; dy <= radius; dy++) {
          for (let dx = -radius; dx <= radius; dx++) {
            if (dx * dx + dy * dy > radius * radius) continue;
            const nx = Math.max(0, Math.min(width - 1, x + dx));
            const ny = Math.max(0, Math.min(height - 1, y + dy));
            const idx = (ny * width + nx) * 4;
            const r = srcData[idx], g = srcData[idx + 1], b = srcData[idx + 2];
            
            for (let pi = 0; pi < palette.length; pi++) {
              const pc = palette[pi];
              if (pc.r === r && pc.g === g && pc.b === b) {
                colorCounts.set(pi, (colorCounts.get(pi) || 0) + 1);
                break;
              }
            }
          }
        }
        
        // For dilation, take the MOST DOMINANT color in neighborhood
        let maxCount = 0;
        let winningIndex = 0;
        for (const [colorIndex, count] of colorCounts.entries()) {
          if (count > maxCount) {
            maxCount = count;
            winningIndex = colorIndex;
          }
        }
        
        const winningColor = palette[winningIndex] || palette[0];
        const idx = (y * width + x) * 4;
        data[idx] = winningColor.r;
        data[idx + 1] = winningColor.g;
        data[idx + 2] = winningColor.b;
      }
    }
    
    return output;
  }, [result]);

  // Remove small isolated regions by flood-fill and merge
  const removeSmallRegions = useCallback((imageData: ImageData, minSize: number = 50): ImageData => {
    if (!result) return imageData;
    
    const palette = result.palette.map(p => p.dominantColor);
    const width = imageData.width;
    const height = imageData.height;
    const output = new ImageData(new Uint8ClampedArray(imageData.data), width, height);
    const data = output.data;
    
    // Track which pixels have been visited
    const visited = new Array(width * height).fill(false);
    
    // Get color index at pixel
    const getColorIndex = (x: number, y: number): number => {
      const idx = (y * width + x) * 4;
      const r = data[idx], g = data[idx + 1], b = data[idx + 2];
      for (let pi = 0; pi < palette.length; pi++) {
        const pc = palette[pi];
        if (pc.r === r && pc.g === g && pc.b === b) return pi;
      }
      return 0;
    };
    
    // Flood fill to find connected region
    const floodFill = (startX: number, startY: number, colorIndex: number): Array<[number, number]> => {
      const region: Array<[number, number]> = [];
      const stack: Array<[number, number]> = [[startX, startY]];
      
      while (stack.length > 0) {
        const [x, y] = stack.pop()!;
        const key = y * width + x;
        
        if (x < 0 || x >= width || y < 0 || y >= height) continue;
        if (visited[key]) continue;
        if (getColorIndex(x, y) !== colorIndex) continue;
        
        visited[key] = true;
        region.push([x, y]);
        
        // 4-connectivity
        stack.push([x + 1, y], [x - 1, y], [x, y + 1], [x, y - 1]);
      }
      
      return region;
    };
    
    // Find dominant neighbor color for a region
    const findNeighborColor = (region: Array<[number, number]>, ownColorIndex: number): number => {
      const neighborCounts = new Map<number, number>();
      
      for (const [x, y] of region) {
        for (const [dx, dy] of [[1, 0], [-1, 0], [0, 1], [0, -1]]) {
          const nx = x + dx, ny = y + dy;
          if (nx < 0 || nx >= width || ny < 0 || ny >= height) continue;
          const neighborColor = getColorIndex(nx, ny);
          if (neighborColor !== ownColorIndex) {
            neighborCounts.set(neighborColor, (neighborCounts.get(neighborColor) || 0) + 1);
          }
        }
      }
      
      let maxCount = 0;
      let bestNeighbor = ownColorIndex;
      for (const [colorIndex, count] of neighborCounts.entries()) {
        if (count > maxCount) {
          maxCount = count;
          bestNeighbor = colorIndex;
        }
      }
      
      return bestNeighbor;
    };
    
    // Process all regions
    const regionsToMerge: Array<{region: Array<[number, number]>, newColorIndex: number}> = [];
    
    for (let y = 0; y < height; y++) {
      for (let x = 0; x < width; x++) {
        const key = y * width + x;
        if (visited[key]) continue;
        
        const colorIndex = getColorIndex(x, y);
        const region = floodFill(x, y, colorIndex);
        
        if (region.length < minSize) {
          const newColorIndex = findNeighborColor(region, colorIndex);
          regionsToMerge.push({ region, newColorIndex });
        }
      }
    }
    
    // Apply merges
    for (const { region, newColorIndex } of regionsToMerge) {
      const newColor = palette[newColorIndex] || palette[0];
      for (const [x, y] of region) {
        const idx = (y * width + x) * 4;
        data[idx] = newColor.r;
        data[idx + 1] = newColor.g;
        data[idx + 2] = newColor.b;
      }
    }
    
    return output;
  }, [result]);

  // Apply multiple smoothing passes with increasing effect
  const applySmoothing = useCallback((imageData: ImageData, passes: number): ImageData => {
    let current = imageData;
    
    // Apply smoothing passes with radius that grows slightly
    for (let i = 0; i < passes; i++) {
      const radius = Math.min(3 + Math.floor(i / 2), 6);
      current = applySmoothingPass(current, radius);
    }
    
    // Apply morphological closing (dilate then smooth) for cleaner edges
    if (passes >= 2) {
      current = applyDilation(current, 2);
      current = applySmoothingPass(current, 3);
    }
    
    // Remove small isolated regions
    if (passes >= 3) {
      const minRegionSize = Math.max(30, passes * 15);
      current = removeSmallRegions(current, minRegionSize);
    }
    
    return current;
  }, [applySmoothingPass, applyDilation, removeSmallRegions]);

  // Apply downsampling to posterized image
  const applyDownsampling = useCallback((posterized: ImageData, size: number): ImageData => {
    if (size <= 1 || !result) return posterized;
    
    const palette = result.palette.map(p => p.dominantColor);
    const width = posterized.width;
    const height = posterized.height;
    const output = new ImageData(
      new Uint8ClampedArray(posterized.data),
      width,
      height
    );
    const data = output.data;
    const srcData = posterized.data;

    // Process each tile
    for (let tileY = 0; tileY < height; tileY += size) {
      for (let tileX = 0; tileX < width; tileX += size) {
        // Count colors in this tile
        const colorCounts = new Map<number, number>();
        
        for (let y = tileY; y < Math.min(tileY + size, height); y++) {
          for (let x = tileX; x < Math.min(tileX + size, width); x++) {
            const idx = (y * width + x) * 4;
            const r = srcData[idx];
            const g = srcData[idx + 1];
            const b = srcData[idx + 2];
            
            // Find which palette color this pixel is
            for (let pi = 0; pi < palette.length; pi++) {
              const pc = palette[pi];
              if (pc.r === r && pc.g === g && pc.b === b) {
                colorCounts.set(pi, (colorCounts.get(pi) || 0) + 1);
                break;
              }
            }
          }
        }
        
        // Find most common color (lowest index wins ties)
        let maxCount = 0;
        let winningIndex = 0;
        for (const [colorIndex, count] of colorCounts.entries()) {
          if (count > maxCount || (count === maxCount && colorIndex < winningIndex)) {
            maxCount = count;
            winningIndex = colorIndex;
          }
        }
        
        const winningColor = palette[winningIndex] || palette[0];
        
        // Fill entire tile with winning color
        for (let y = tileY; y < Math.min(tileY + size, height); y++) {
          for (let x = tileX; x < Math.min(tileX + size, width); x++) {
            const idx = (y * width + x) * 4;
            data[idx] = winningColor.r;
            data[idx + 1] = winningColor.g;
            data[idx + 2] = winningColor.b;
          }
        }
      }
    }
    
    return output;
  }, [result]);

  // Draw highlighted image on modal canvas when modal opens
  useEffect(() => {
    if (!modalOpen || highlightedColorIndex === null || !result || !originalImageData) return;
    
    // Wait for modal canvas to be in DOM
    setTimeout(() => {
      const modalCanvas = modalCanvasRef.current;
      if (!modalCanvas) return;
      
      const ctx = modalCanvas.getContext('2d');
      if (!ctx) return;

      const color = result.palette[highlightedColorIndex].dominantColor;
      const palette = result.palette.map(p => p.dominantColor);
      
      // Determine source image based on current mode
      let sourceImageData = originalImageData;
      const usePosterized = posterizeMode && posterizedImageData;
      
      if (usePosterized) {
        // Start with posterized image
        let processed = posterizedImageData;
        
        // Apply smoothing if enabled
        if (smoothingPasses > 0) {
          processed = applySmoothing(processed, smoothingPasses);
        }
        
        // Apply downsampling if needed
        if (tileSize > 1) {
          processed = applyDownsampling(processed, tileSize);
        }
        
        sourceImageData = processed;
      }
      
      // Set modal canvas size (larger than main canvas)
      const maxWidth = 800;
      const maxHeight = 600;
      let width = sourceImageData.width;
      let height = sourceImageData.height;
      
      // Scale up for modal
      const scale = Math.min(maxWidth / width, maxHeight / height, 2);
      width = Math.round(width * scale);
      height = Math.round(height * scale);

      modalCanvas.width = width;
      modalCanvas.height = height;
      
      // Create temp canvas with source size
      const tempCanvas = document.createElement('canvas');
      tempCanvas.width = sourceImageData.width;
      tempCanvas.height = sourceImageData.height;
      const tempCtx = tempCanvas.getContext('2d');
      if (!tempCtx) return;
      
      // Put source image data and highlight
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
          // Matching pixels: use the selected color
          data[i] = color.r;
          data[i + 1] = color.g;
          data[i + 2] = color.b;
        } else {
          // Non-matching pixels: color white
          data[i] = 255;
          data[i + 1] = 255;
          data[i + 2] = 255;
        }
      }

      tempCtx.putImageData(imageData, 0, 0);
      
      // Draw scaled to modal canvas (disable smoothing for posterized to keep sharp tiles)
      ctx.imageSmoothingEnabled = !usePosterized;
      ctx.drawImage(tempCanvas, 0, 0, width, height);
    }, 50);
  }, [modalOpen, highlightedColorIndex, result, originalImageData, posterizeMode, posterizedImageData, tileSize, smoothingPasses, applySmoothing, applyDownsampling]);

  // Export modal content to PDF
  const handlePrint = useCallback(async () => {
    if (!result || highlightedColorIndex === null || !modalCanvasRef.current || !originalImageData) return;
    
    const item = result.palette[highlightedColorIndex];
    
    // Get the modified canvas image as base64
    const canvas = modalCanvasRef.current;
    const imageData = canvas.toDataURL('image/png');
    
    // Create original image as base64 (render the unmodified original to a temp canvas)
    const origCanvas = document.createElement('canvas');
    origCanvas.width = originalImageData.width;
    origCanvas.height = originalImageData.height;
    const origCtx = origCanvas.getContext('2d');
    let originalImageBase64 = '';
    if (origCtx) {
      origCtx.putImageData(originalImageData, 0, 0);
      originalImageBase64 = origCanvas.toDataURL('image/png');
    }
    
    // Build PDF export data
    const pdfDataRaw: {
      colorIndex: number;
      imageData: string;
      originalImageData: string;
      targetHex: string;
      targetRGB: number[];
      resultHex: string;
      mixingRecipe: Array<{
        name: string;
        brand: string;
        series: number;
        pigments: string;
        opacity: string;
        hex: string;
        rgb: number[];
        parts: number;
      }>;
      totalParts: number;
    } = {
      colorIndex: highlightedColorIndex + 1, // 1-based for display
      imageData: imageData,
      originalImageData: originalImageBase64,
      targetHex: item.dominantColor.hex,
      targetRGB: [item.dominantColor.r, item.dominantColor.g, item.dominantColor.b],
      resultHex: '',
      mixingRecipe: [],
      totalParts: 0,
    };
    
    // Calculate mixing recipe data
    if (item.mixingRecipe && item.mixingRecipe.length > 0) {
      const totalParts = item.mixingRecipe.reduce((sum, p) => sum + p.parts, 0);
      pdfDataRaw.totalParts = totalParts;
      
      // Calculate mixed color
      let mixR = 0, mixG = 0, mixB = 0;
      item.mixingRecipe.forEach(part => {
        mixR += part.paint.rgb[0] * part.parts;
        mixG += part.paint.rgb[1] * part.parts;
        mixB += part.paint.rgb[2] * part.parts;
      });
      mixR = Math.round(mixR / totalParts);
      mixG = Math.round(mixG / totalParts);
      mixB = Math.round(mixB / totalParts);
      pdfDataRaw.resultHex = `#${mixR.toString(16).padStart(2, '0')}${mixG.toString(16).padStart(2, '0')}${mixB.toString(16).padStart(2, '0')}`.toUpperCase();
      
      // Build paint parts array
      pdfDataRaw.mixingRecipe = item.mixingRecipe.map(part => ({
        name: part.paint.name,
        brand: part.paint.brand,
        series: part.paint.series,
        pigments: part.paint.pigments,
        opacity: part.paint.opacity,
        hex: part.paint.hex,
        rgb: part.paint.rgb,
        parts: part.parts,
      }));
    }
    
    try {
      const pdfData = main.PDFExportData.createFrom(pdfDataRaw);
      await ExportColorDetailPDF(pdfData);
    } catch (err) {
      console.error('Failed to export PDF:', err);
    }
  }, [result, highlightedColorIndex, originalImageData]);

  // ============================================
  // PDF Export Context & Handlers
  // ============================================

  // Build the context object for PDF handlers
  const getPdfContext = useCallback((): PdfExportContext => ({
    result,
    originalImageData,
    posterizedImageData,
    posterizeMode,
    smoothingPasses,
    tileSize,
    canvasRef,
  }), [result, originalImageData, posterizedImageData, posterizeMode, smoothingPasses, tileSize]);

  // PDF action handlers (1-5)
  const handlePdfAction1 = useCallback(() => {
    exportComparisonPdf(getPdfContext());
  }, [getPdfContext]);

  const handlePdfAction2 = useCallback(() => {
    exportPdf2(getPdfContext());
  }, [getPdfContext]);

  const handlePdfAction3 = useCallback(() => {
    exportPdf3(getPdfContext());
  }, [getPdfContext]);

  const handlePdfAction4 = useCallback(() => {
    exportPdf4(getPdfContext());
  }, [getPdfContext]);

  const handlePdfAction5 = useCallback(() => {
    exportPdf5(getPdfContext());
  }, [getPdfContext]);

  // Keyboard navigation for cycling through colors and hotkeys
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Close modals on Escape
      if (e.key === 'Escape') {
        if (modalOpen) {
          e.preventDefault();
          setModalOpen(false);
          return;
        }
      }
      
      // Hotkey for PDF export: Cmd+o
      if ((e.key === 'o' || e.key === 'O') && e.metaKey) {
        e.preventDefault();
        if (modalOpen) {
          // Export color detail PDF from color modal
          handlePrint();
        } else {
          // Export comparison PDF
          handlePdfAction1();
        }
        return;
      }

      // PDF action hotkeys: Cmd+1 through Cmd+5
      if (e.metaKey && !e.shiftKey && !e.altKey) {
        if (e.key === '1') {
          e.preventDefault();
          handlePdfAction1();
          return;
        }
        if (e.key === '2') {
          e.preventDefault();
          handlePdfAction2();
          return;
        }
        if (e.key === '3') {
          e.preventDefault();
          handlePdfAction3();
          return;
        }
        if (e.key === '4') {
          e.preventDefault();
          handlePdfAction4();
          return;
        }
        if (e.key === '5') {
          e.preventDefault();
          handlePdfAction5();
          return;
        }
      }
      
      // Hotkeys for nColors: Cmd+n (increase), Shift+n (decrease)
      if (e.key === 'n' || e.key === 'N') {
        if (e.metaKey && !e.shiftKey) {
          e.preventDefault();
          const newValue = Math.min(40, nColors + 1);
          setNColorsState(newValue);
          SetNColors(newValue);
          if (result) {
            if (reprocessTimeoutRef.current) {
              clearTimeout(reprocessTimeoutRef.current);
            }
            reprocessTimeoutRef.current = window.setTimeout(() => {
              ReprocessImage();
              reprocessTimeoutRef.current = null;
            }, 300);
          }
          return;
        } else if (e.metaKey && e.shiftKey) {
          e.preventDefault();
          const newValue = Math.max(2, nColors - 1);
          setNColorsState(newValue);
          SetNColors(newValue);
          if (result) {
            if (reprocessTimeoutRef.current) {
              clearTimeout(reprocessTimeoutRef.current);
            }
            reprocessTimeoutRef.current = window.setTimeout(() => {
              ReprocessImage();
              reprocessTimeoutRef.current = null;
            }, 300);
          }
          return;
        }
      }
      
      // Hotkeys for tileSize: Cmd+t (cycle forward), Cmd+Shift+t (cycle backward)
      if (e.key === 't' || e.key === 'T') {
        if (e.metaKey && !e.shiftKey) {
          e.preventDefault();
          const newSize = tileSize >= 16 ? 1 : tileSize + 1;
          setTileSize(newSize);
          return;
        } else if (e.metaKey && e.shiftKey) {
          e.preventDefault();
          const newSize = tileSize <= 1 ? 16 : tileSize - 1;
          setTileSize(newSize);
          return;
        }
      }
      
      // Hotkey for posterize: Cmd+p
      if ((e.key === 'p' || e.key === 'P') && e.metaKey && !e.shiftKey) {
        e.preventDefault();
        setPosterizeMode(!posterizeMode);
        return;
      }
      
      // Hotkeys for smoothing: Cmd+s (cycle forward), Cmd+Shift+s (cycle backward)
      if (e.key === 's' || e.key === 'S') {
        if (e.metaKey && !e.shiftKey) {
          e.preventDefault();
          const newPasses = smoothingPasses >= 5 ? 0 : smoothingPasses + 1;
          setSmoothingPasses(newPasses);
          return;
        } else if (e.metaKey && e.shiftKey) {
          e.preventDefault();
          const newPasses = smoothingPasses <= 0 ? 5 : smoothingPasses - 1;
          setSmoothingPasses(newPasses);
          return;
        }
      }
      
      // Hotkey for aspect ratio: Cmd+a (cycle)
      if ((e.key === 'a' || e.key === 'A') && e.metaKey) {
        e.preventDefault();
        const ratios: AspectRatioType[] = ['original', 'landscape', 'portrait', 'square'];
        const idx = ratios.indexOf(aspectRatio);
        setAspectRatio(ratios[(idx + 1) % ratios.length]);
        return;
      }
      
      if (!result || result.palette.length === 0) return;
      
      const paletteLength = result.palette.length;
      
      if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
        e.preventDefault();
        setHighlightedColorIndex(prev => {
          if (prev === null) {
            // Resume from last position, move to next
            const newIndex = (lastColorIndex + 1) % paletteLength;
            setLastColorIndex(newIndex);
            return newIndex;
          }
          const newIndex = (prev + 1) % paletteLength;
          setLastColorIndex(newIndex);
          return newIndex;
        });
      } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
        e.preventDefault();
        setHighlightedColorIndex(prev => {
          if (prev === null) {
            // Resume from last position, move to previous
            const newIndex = (lastColorIndex - 1 + paletteLength) % paletteLength;
            setLastColorIndex(newIndex);
            return newIndex;
          }
          const newIndex = (prev - 1 + paletteLength) % paletteLength;
          setLastColorIndex(newIndex);
          return newIndex;
        });
      } else if (e.key === 'Enter' && highlightedColorIndex !== null) {
        e.preventDefault();
        setModalOpen(true);
      } else if (e.key === 'Escape' && highlightedColorIndex !== null) {
        e.preventDefault();
        // Restore base canvas (original or posterized+downsampled)
        const canvas = canvasRef.current;
        if (canvas && originalImageData) {
          const ctx = canvas.getContext('2d');
          if (ctx) {
            if (posterizeMode && posterizedImageData && result) {
              // Apply downsampling to posterized image
              const palette = result.palette.map(p => p.dominantColor);
              const width = posterizedImageData.width;
              const height = posterizedImageData.height;
              
              if (tileSize <= 1) {
                ctx.putImageData(posterizedImageData, 0, 0);
              } else {
                const output = new ImageData(
                  new Uint8ClampedArray(posterizedImageData.data),
                  width,
                  height
                );
                const data = output.data;
                const srcData = posterizedImageData.data;

                for (let tileY = 0; tileY < height; tileY += tileSize) {
                  for (let tileX = 0; tileX < width; tileX += tileSize) {
                    const colorCounts = new Map<number, number>();
                    
                    for (let y = tileY; y < Math.min(tileY + tileSize, height); y++) {
                      for (let x = tileX; x < Math.min(tileX + tileSize, width); x++) {
                        const idx = (y * width + x) * 4;
                        const r = srcData[idx];
                        const g = srcData[idx + 1];
                        const b = srcData[idx + 2];
                        
                        for (let pi = 0; pi < palette.length; pi++) {
                          const pc = palette[pi];
                          if (pc.r === r && pc.g === g && pc.b === b) {
                            colorCounts.set(pi, (colorCounts.get(pi) || 0) + 1);
                            break;
                          }
                        }
                      }
                    }
                    
                    let maxCount = 0;
                    let winningIndex = 0;
                    for (const [colorIndex, count] of colorCounts.entries()) {
                      if (count > maxCount || (count === maxCount && colorIndex < winningIndex)) {
                        maxCount = count;
                        winningIndex = colorIndex;
                      }
                    }
                    
                    const winningColor = palette[winningIndex] || palette[0];
                    
                    for (let y = tileY; y < Math.min(tileY + tileSize, height); y++) {
                      for (let x = tileX; x < Math.min(tileX + tileSize, width); x++) {
                        const idx = (y * width + x) * 4;
                        data[idx] = winningColor.r;
                        data[idx + 1] = winningColor.g;
                        data[idx + 2] = winningColor.b;
                      }
                    }
                  }
                }
                ctx.putImageData(output, 0, 0);
              }
            } else {
              ctx.putImageData(originalImageData, 0, 0);
            }
          }
        }
        setHighlightedColorIndex(null);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [result, modalOpen, highlightedColorIndex, originalImageData, lastColorIndex, posterizeMode, posterizedImageData, tileSize, nColors, setTileSize, setPosterizeMode, smoothingPasses, setSmoothingPasses, aspectRatio, setAspectRatio, handlePrint, handlePdfAction1, handlePdfAction2, handlePdfAction3, handlePdfAction4, handlePdfAction5]);

  // Update canvas highlighting when highlightedColorIndex changes via keyboard
  useEffect(() => {
    if (!result || highlightedColorIndex === null || !canvasRef.current || !originalImageData) return;
    
    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const color = result.palette[highlightedColorIndex].dominantColor;
    const palette = result.palette.map(p => p.dominantColor);
    const usePosterized = posterizeMode && posterizedImageData;
    
    // Determine base image (with downsampling if needed)
    let baseImageData = originalImageData;
    if (usePosterized) {
      if (tileSize > 1) {
        // Apply downsampling
        const width = posterizedImageData.width;
        const height = posterizedImageData.height;
        const output = new ImageData(
          new Uint8ClampedArray(posterizedImageData.data),
          width,
          height
        );
        const outData = output.data;
        const srcData = posterizedImageData.data;

        for (let tileY = 0; tileY < height; tileY += tileSize) {
          for (let tileX = 0; tileX < width; tileX += tileSize) {
            const colorCounts = new Map<number, number>();
            
            for (let y = tileY; y < Math.min(tileY + tileSize, height); y++) {
              for (let x = tileX; x < Math.min(tileX + tileSize, width); x++) {
                const idx = (y * width + x) * 4;
                const pr = srcData[idx];
                const pg = srcData[idx + 1];
                const pb = srcData[idx + 2];
                
                for (let pi = 0; pi < palette.length; pi++) {
                  const pc = palette[pi];
                  if (pc.r === pr && pc.g === pg && pc.b === pb) {
                    colorCounts.set(pi, (colorCounts.get(pi) || 0) + 1);
                    break;
                  }
                }
              }
            }
            
            let maxCount = 0;
            let winningIndex = 0;
            for (const [colorIndex, count] of colorCounts.entries()) {
              if (count > maxCount || (count === maxCount && colorIndex < winningIndex)) {
                maxCount = count;
                winningIndex = colorIndex;
              }
            }
            
            const winningColor = palette[winningIndex] || palette[0];
            
            for (let y = tileY; y < Math.min(tileY + tileSize, height); y++) {
              for (let x = tileX; x < Math.min(tileX + tileSize, width); x++) {
                const idx = (y * width + x) * 4;
                outData[idx] = winningColor.r;
                outData[idx + 1] = winningColor.g;
                outData[idx + 2] = winningColor.b;
              }
            }
          }
        }
        baseImageData = output;
      } else {
        baseImageData = posterizedImageData;
      }
    }
    
    // Restore base, then highlight
    ctx.putImageData(baseImageData, 0, 0);
    const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    const data = imageData.data;

    for (let i = 0; i < data.length; i += 4) {
      const r = data[i];
      const g = data[i + 1];
      const b = data[i + 2];

      // For posterized, use exact match; for original, use threshold
      const isMatch = usePosterized
        ? (r === color.r && g === color.g && b === color.b)
        : (colorDistance(r, g, b, color.r, color.g, color.b) < COLOR_THRESHOLD);
      
      if (isMatch) {
        data[i] = 255;
        data[i + 1] = 0;
        data[i + 2] = 0;
      }
    }

    ctx.putImageData(imageData, 0, 0);
  }, [highlightedColorIndex, result, originalImageData, posterizeMode, posterizedImageData, tileSize]);

  // Draw image on canvas when result changes
  useEffect(() => {
    if (result && canvasRef.current) {
      const canvas = canvasRef.current;
      const ctx = canvas.getContext('2d');
      if (!ctx) return;

      const img = new Image();
      img.onload = () => {
        // Set canvas size to match image (with max constraints)
        const maxWidth = 600;
        const maxHeight = 400;
        let width = img.width;
        let height = img.height;

        if (width > maxWidth) {
          height = (maxWidth / width) * height;
          width = maxWidth;
        }
        if (height > maxHeight) {
          width = (maxHeight / height) * width;
          height = maxHeight;
        }

        canvas.width = width;
        canvas.height = height;
        ctx.drawImage(img, 0, 0, width, height);
        imageRef.current = img;
        
        // Store original image data for restoration
        setOriginalImageData(ctx.getImageData(0, 0, width, height));
      };
      img.src = result.imageData;
    }
  }, [result]);

  // Re-apply posterize mode when new image is loaded (if mode was active)
  useEffect(() => {
    if (!originalImageData || !result || !posterizeMode) return;
    
    const canvas = canvasRef.current;
    if (!canvas) return;
    
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    
    // Generate new posterized image for new data
    const palette = result.palette.map(p => p.dominantColor);
    const imageData = new ImageData(
      new Uint8ClampedArray(originalImageData.data),
      originalImageData.width,
      originalImageData.height
    );
    const data = imageData.data;

    for (let i = 0; i < data.length; i += 4) {
      const r = data[i];
      const g = data[i + 1];
      const b = data[i + 2];

      let minDist = Infinity;
      let closestColor = palette[0];
      for (const color of palette) {
        const dist = (r - color.r) ** 2 + (g - color.g) ** 2 + (b - color.b) ** 2;
        if (dist < minDist) {
          minDist = dist;
          closestColor = color;
        }
      }

      data[i] = closestColor.r;
      data[i + 1] = closestColor.g;
      data[i + 2] = closestColor.b;
    }
    
    setPosterizedImageData(imageData);
    
    // Apply downsampling if needed and draw
    if (tileSize > 1) {
      const width = imageData.width;
      const height = imageData.height;
      const output = new ImageData(
        new Uint8ClampedArray(imageData.data),
        width,
        height
      );
      const outData = output.data;
      const srcData = imageData.data;

      for (let tileY = 0; tileY < height; tileY += tileSize) {
        for (let tileX = 0; tileX < width; tileX += tileSize) {
          const colorCounts = new Map<number, number>();
          
          for (let y = tileY; y < Math.min(tileY + tileSize, height); y++) {
            for (let x = tileX; x < Math.min(tileX + tileSize, width); x++) {
              const idx = (y * width + x) * 4;
              const pr = srcData[idx];
              const pg = srcData[idx + 1];
              const pb = srcData[idx + 2];
              
              for (let pi = 0; pi < palette.length; pi++) {
                const pc = palette[pi];
                if (pc.r === pr && pc.g === pg && pc.b === pb) {
                  colorCounts.set(pi, (colorCounts.get(pi) || 0) + 1);
                  break;
                }
              }
            }
          }
          
          let maxCount = 0;
          let winningIndex = 0;
          for (const [colorIndex, count] of colorCounts.entries()) {
            if (count > maxCount || (count === maxCount && colorIndex < winningIndex)) {
              maxCount = count;
              winningIndex = colorIndex;
            }
          }
          
          const winningColor = palette[winningIndex] || palette[0];
          
          for (let y = tileY; y < Math.min(tileY + tileSize, height); y++) {
            for (let x = tileX; x < Math.min(tileX + tileSize, width); x++) {
              const idx = (y * width + x) * 4;
              outData[idx] = winningColor.r;
              outData[idx + 1] = winningColor.g;
              outData[idx + 2] = winningColor.b;
            }
          }
        }
      }
      ctx.putImageData(output, 0, 0);
    } else {
      ctx.putImageData(imageData, 0, 0);
    }
  }, [originalImageData, result, posterizeMode, tileSize]);

  // Calculate color distance (simple Euclidean)
  const colorDistance = (r1: number, g1: number, b1: number, r2: number, g2: number, b2: number): number => {
    return Math.sqrt((r1 - r2) ** 2 + (g1 - g2) ** 2 + (b1 - b2) ** 2);
  };

  // Generate posterized image data
  const generatePosterizedImage = useCallback(() => {
    if (!originalImageData || !result || result.palette.length === 0) return null;
    
    const palette = result.palette.map(p => p.dominantColor);
    const imageData = new ImageData(
      new Uint8ClampedArray(originalImageData.data),
      originalImageData.width,
      originalImageData.height
    );
    const data = imageData.data;

    for (let i = 0; i < data.length; i += 4) {
      const r = data[i];
      const g = data[i + 1];
      const b = data[i + 2];

      // Find closest palette color
      let minDist = Infinity;
      let closestColor = palette[0];
      for (const color of palette) {
        const dist = (r - color.r) ** 2 + (g - color.g) ** 2 + (b - color.b) ** 2;
        if (dist < minDist) {
          minDist = dist;
          closestColor = color;
        }
      }

      data[i] = closestColor.r;
      data[i + 1] = closestColor.g;
      data[i + 2] = closestColor.b;
    }

    return imageData;
  }, [originalImageData, result]);

  // React to tileSize changes and update canvas if in posterize mode
  // This allows keyboard shortcuts to trigger canvas updates
  const prevTileSizeRef = useRef(tileSize);
  useEffect(() => {
    if (prevTileSizeRef.current === tileSize) return;
    prevTileSizeRef.current = tileSize;
    
    const canvas = canvasRef.current;
    if (!canvas || !posterizedImageData || !posterizeMode) return;
    
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    
    setHighlightedColorIndex(null);
    // Apply smoothing then downsampling
    let processed = posterizedImageData;
    if (smoothingPasses > 0) {
      processed = applySmoothing(processed, smoothingPasses);
    }
    const downsampled = applyDownsampling(processed, tileSize);
    ctx.putImageData(downsampled, 0, 0);
  }, [tileSize, posterizedImageData, posterizeMode, applyDownsampling, smoothingPasses, applySmoothing]);

  // React to smoothingPasses changes and update canvas if in posterize mode
  const prevSmoothingPassesRef = useRef(smoothingPasses);
  useEffect(() => {
    if (prevSmoothingPassesRef.current === smoothingPasses) return;
    prevSmoothingPassesRef.current = smoothingPasses;
    
    const canvas = canvasRef.current;
    if (!canvas || !posterizedImageData || !posterizeMode) return;
    
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    
    setHighlightedColorIndex(null);
    // Apply smoothing then downsampling
    let processed = posterizedImageData;
    if (smoothingPasses > 0) {
      processed = applySmoothing(processed, smoothingPasses);
    }
    const downsampled = applyDownsampling(processed, tileSize);
    ctx.putImageData(downsampled, 0, 0);
  }, [smoothingPasses, posterizedImageData, posterizeMode, applySmoothing, applyDownsampling, tileSize]);

  // React to posterizeMode changes and update canvas
  // This allows keyboard shortcuts to trigger canvas updates
  const prevPosterizeModeRef = useRef(posterizeMode);
  useEffect(() => {
    if (prevPosterizeModeRef.current === posterizeMode) return;
    prevPosterizeModeRef.current = posterizeMode;
    
    const canvas = canvasRef.current;
    if (!canvas || !originalImageData) return;
    
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    
    setHighlightedColorIndex(null);
    
    if (posterizeMode) {
      // Switching to posterize mode
      let posterized = posterizedImageData;
      if (!posterized) {
        posterized = generatePosterizedImage();
        if (posterized) {
          setPosterizedImageData(posterized);
        }
      }
      if (posterized) {
        // Apply smoothing then downsampling
        let processed = posterized;
        if (smoothingPasses > 0) {
          processed = applySmoothing(processed, smoothingPasses);
        }
        const downsampled = applyDownsampling(processed, tileSize);
        ctx.putImageData(downsampled, 0, 0);
      }
    } else {
      // Switching back to original
      ctx.putImageData(originalImageData, 0, 0);
    }
  }, [posterizeMode, posterizedImageData, originalImageData, generatePosterizedImage, tileSize, applyDownsampling, smoothingPasses, applySmoothing]);

  // Handle tile size cycle
  const handleTileSizeCycle = useCallback(() => {
    const newSize = tileSize >= 16 ? 1 : tileSize + 1;
    setTileSize(newSize);
    
    // Canvas update handled by useEffect
  }, [tileSize, setTileSize]);

  // Handle posterize mode toggle
  const handlePosterizeToggle = useCallback(() => {
    if (!originalImageData) return;
    
    setPosterizeMode(!posterizeMode);
    // Canvas update handled by useEffect
  }, [posterizeMode, originalImageData, setPosterizeMode]);

  // Handle clicking on a dominant color swatch to highlight/unhighlight pixels
  const handlePaletteColorClick = useCallback((idx: number, color: ColorInfo) => {
    const canvas = canvasRef.current;
    if (!canvas || !originalImageData) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Always track the last selected color index
    setLastColorIndex(idx);

    // Determine which base image to use (with downsampling if in posterize mode)
    let baseImageData = originalImageData;
    if (posterizeMode && posterizedImageData) {
      baseImageData = applyDownsampling(posterizedImageData, tileSize);
    }

    if (highlightedColorIndex === idx) {
      // Restore base image (original or posterized+downsampled)
      ctx.putImageData(baseImageData, 0, 0);
      setHighlightedColorIndex(null);
    } else {
      // First restore base image, then highlight the new color
      ctx.putImageData(baseImageData, 0, 0);
      
      // Get current image data
      const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
      const data = imageData.data;

      // Iterate through pixels and highlight matching ones in red
      for (let i = 0; i < data.length; i += 4) {
        const r = data[i];
        const g = data[i + 1];
        const b = data[i + 2];

        const dist = colorDistance(r, g, b, color.r, color.g, color.b);
        if (dist < COLOR_THRESHOLD) {
          // Set to bright red
          data[i] = 255;     // R
          data[i + 1] = 0;   // G
          data[i + 2] = 0;   // B
          // Alpha stays the same
        }
      }

      ctx.putImageData(imageData, 0, 0);
      setHighlightedColorIndex(idx);
    }
  }, [highlightedColorIndex, originalImageData, posterizeMode, posterizedImageData, tileSize, applyDownsampling]);

  // Get match badge color based on rating
  const getMatchBadgeClass = (rating: string): string => {
    switch (rating) {
      case 'Perfect Match':
      case 'Excellent Match':
        return 'badge-excellent';
      case 'Good Alternative':
        return 'badge-good';
      case 'Approximate':
        return 'badge-approximate';
      default:
        return 'badge-none';
    }
  };

  return (
    <div id="App">
      {/* Header */}
      <header className="app-header">
        <h1>🎨 AcrylicMaster</h1>
        <p>Image Color Palette & Paint Matcher</p>
      </header>

      {/* Drop Zone */}
      <div className="drop-zone" style={{ '--wails-drop-target': 'drop' } as React.CSSProperties}>
        {!result && !isProcessing && (
          <div className="drop-prompt">
            <div className="drop-icon">📷</div>
            <h2>Drop an image here</h2>
            <p>Supported formats: JPG, PNG, GIF</p>
            <div className="ncolors-slider">
              <label htmlFor="ncolors-input">Colors to extract: <strong>{nColors}</strong></label>
              <input
                id="ncolors-input"
                type="range"
                min="2"
                max="40"
                value={nColors}
                onChange={handleNColorsChange}
              />
            </div>
          </div>
        )}

        {isProcessing && (
          <div className="processing">
            <div className="spinner"></div>
            <p>{progressMessage}</p>
          </div>
        )}

        {error && (
          <div className="error">
            <p>❌ {error}</p>
          </div>
        )}

        {result && !isProcessing && (
          <div className="content">
            {/* Image Canvas */}
            <div className={`canvas-container aspect-${aspectRatio}`}>
              <div className="canvas-wrapper">
                <canvas
                  ref={canvasRef}
                  className="image-canvas"
                  onClick={handlePdfAction1}
                  style={{ cursor: 'pointer' }}
                  title="Click to export PDF"
                />
                <div className="pdf-action-buttons">
                  <button className="pdf-action-btn" onClick={handlePdfAction1} title="Export PDF (⌘1)">1</button>
                  <button className="pdf-action-btn" onClick={handlePdfAction2} title="PDF Action 2 (⌘2)">2</button>
                  <button className="pdf-action-btn" onClick={handlePdfAction3} title="PDF Action 3 (⌘3)">3</button>
                  <button className="pdf-action-btn" onClick={handlePdfAction4} title="PDF Action 4 (⌘4)">4</button>
                  <button className="pdf-action-btn" onClick={handlePdfAction5} title="PDF Action 5 (⌘5)">5</button>
                </div>
              </div>
              <div className="canvas-controls">
                <div className="canvas-controls-right">
                  <div className="ncolors-slider compact">
                    <label htmlFor="ncolors-slider">Colors: <strong>{nColors}</strong></label>
                    <input
                      id="ncolors-slider"
                      type="range"
                      min="2"
                      max="40"
                      value={nColors}
                      onChange={handleNColorsChange}
                      title="Adjust and drop a new image to re-extract"
                    />
                  </div>
                  <button 
                    className={`posterize-toggle ${posterizeMode ? 'active' : ''}`}
                    onClick={handlePosterizeToggle}
                    title={posterizeMode ? "Show original image" : "Show image using only palette colors"}
                  >
                    {posterizeMode ? '🎨 Original' : '🖼️ Posterize'}
                  </button>
                  <button 
                    className={`tile-size-toggle ${tileSize > 1 ? 'active' : ''}`}
                    onClick={handleTileSizeCycle}
                    title="Cycle tile size for paintable blocks"
                  >
                    🔲 Tiles: {tileSize}x
                  </button>
                  <button 
                    className={`smoothing-toggle ${smoothingPasses > 0 ? 'active' : ''}`}
                    onClick={() => setSmoothingPasses(smoothingPasses >= 10 ? 0 : smoothingPasses + 1)}
                    title="Paint-by-numbers mode: smooth regions to remove salt-and-pepper noise"
                  >
                    🖌️ Smooth: {smoothingPasses}
                  </button>
                  <button 
                    className={`aspect-toggle ${aspectRatio !== 'original' ? 'active' : ''}`}
                    onClick={() => {
                      const ratios: AspectRatioType[] = ['original', 'landscape', 'portrait', 'square'];
                      const idx = ratios.indexOf(aspectRatio);
                      setAspectRatio(ratios[(idx + 1) % ratios.length]);
                    }}
                    title="Cycle aspect ratio: Original → Landscape (16:9) → Portrait (9:16) → Square"
                  >
                    📐 {aspectRatio === 'original' ? 'Original' : aspectRatio === 'landscape' ? '16:9' : aspectRatio === 'portrait' ? '9:16' : '1:1'}
                  </button>
                </div>
              </div>
            </div>

            {/* Results Panel */}
            <div className="results-panel">
              {/* Dominant Colors Palette */}
              <section className="palette-section">
                <h3>Dominant Colors <span className="hint">(click to highlight in image)</span></h3>
                <div className="palette-grid">
                  {result.palette.map((item, idx) => {
                    // Calculate luminance for text color
                    const r = item.dominantColor.r;
                    const g = item.dominantColor.g;
                    const b = item.dominantColor.b;
                    const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
                    const textColor = luminance > 0.5 ? '#000000' : '#FFFFFF';
                    
                    return (
                    <div 
                      key={idx} 
                      className={`palette-item ${highlightedColorIndex === idx ? 'highlighted' : ''}`}
                      onClick={() => handlePaletteColorClick(idx, item.dominantColor)}
                    >
                      <div
                        className="color-swatch clickable"
                        style={{ backgroundColor: item.dominantColor.hex, color: textColor }}
                        title={`${item.dominantColor.hex} - Click to highlight these pixels`}
                      >
                        <span className="swatch-hex">{item.dominantColor.hex}</span>
                      </div>
                    </div>
                  )})}
                </div>
              </section>

              {/* Shopping List */}
              <section className="shopping-list">
                <h3>🛒 Shopping List</h3>
                <div className="shopping-scroll">
                  {result.palette.map((item, idx) => {
                    // Calculate the expected mixed color
                    let mixedColor = item.dominantColor.hex;
                    let mixedHex = item.dominantColor.hex;
                    const totalParts = item.mixingRecipe?.reduce((sum, p) => sum + p.parts, 0) || 1;
                    
                    if (item.mixingRecipe && item.mixingRecipe.length > 0) {
                      let mixR = 0, mixG = 0, mixB = 0;
                      item.mixingRecipe.forEach(part => {
                        mixR += part.paint.rgb[0] * part.parts;
                        mixG += part.paint.rgb[1] * part.parts;
                        mixB += part.paint.rgb[2] * part.parts;
                      });
                      mixR = Math.round(mixR / totalParts);
                      mixG = Math.round(mixG / totalParts);
                      mixB = Math.round(mixB / totalParts);
                      mixedColor = `rgb(${mixR}, ${mixG}, ${mixB})`;
                      mixedHex = `#${mixR.toString(16).padStart(2, '0')}${mixG.toString(16).padStart(2, '0')}${mixB.toString(16).padStart(2, '0')}`.toUpperCase();
                    }

                    return (
                    <div 
                      key={idx} 
                      className={`recipe-row ${highlightedColorIndex === idx ? 'highlighted' : ''}`}
                      onClick={() => handlePaletteColorClick(idx, item.dominantColor)}
                    >
                      {/* Target → Expected */}
                      <div className="color-comparison">
                        <div className="color-block target">
                          <div className="swatch" style={{ backgroundColor: item.dominantColor.hex }} />
                          <span className="hex-label">{item.dominantColor.hex}</span>
                        </div>
                        <span className="arrow">→</span>
                        <div className="color-block expected">
                          <div className="swatch" style={{ backgroundColor: mixedColor }} />
                          <span className="hex-label">{mixedHex}</span>
                        </div>
                        <span className="equals">=</span>
                      </div>
                      
                      {/* Proportional mix bar */}
                      <div className="mix-bar">
                        {item.mixingRecipe && item.mixingRecipe.map((part, partIdx) => {
                          const widthPercent = (part.parts / totalParts) * 100;
                          // Calculate luminance to determine text color
                          const r = part.paint.rgb[0];
                          const g = part.paint.rgb[1];
                          const b = part.paint.rgb[2];
                          const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
                          const textColor = luminance > 0.5 ? '#000000' : '#FFFFFF';
                          return (
                            <div 
                              key={partIdx}
                              className="mix-segment"
                              style={{ 
                                width: `${widthPercent}%`,
                                backgroundColor: part.paint.hex,
                                color: textColor
                              }}
                              title={`${part.paint.name}: ${part.parts} parts`}
                            >
                              <span className="segment-label">
                                {part.paint.name.length > 12 && widthPercent < 40 
                                  ? part.paint.name.substring(0, 10) + '…' 
                                  : part.paint.name}
                                <strong> ({part.parts})</strong>
                              </span>
                            </div>
                          );
                        })}
                      </div>
                      
                      {highlightedColorIndex === idx && <span className="highlight-badge">VIEWING</span>}
                    </div>
                  )})}
                </div>
              </section>
            </div>
          </div>
        )}
      </div>

      {/* Detail Modal */}
      {modalOpen && highlightedColorIndex !== null && result && (
        <div className="modal-overlay" onClick={() => setModalOpen(false)}>
          <div className="modal-content" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h2>🎨 Color Detail</h2>
              <div className="modal-actions">
                <button className="print-btn" onClick={handlePrint}>� Export PDF</button>
                <button className="close-btn" onClick={() => setModalOpen(false)}>✕</button>
              </div>
            </div>
            
            <div className="modal-body">
              {/* Left side: Image with highlights */}
              <div className="modal-image-section">
                <canvas ref={modalCanvasRef} className="modal-canvas" />
                <p className="modal-image-hint">Highlighted pixels shown in red</p>
              </div>
              
              {/* Right side: Color details */}
              <div className="modal-details-section">
                {(() => {
                  const item = result.palette[highlightedColorIndex];
                  const totalParts = item.mixingRecipe?.reduce((sum, p) => sum + p.parts, 0) || 1;
                  
                  // Calculate mixed color
                  let mixedHex = item.dominantColor.hex;
                  if (item.mixingRecipe && item.mixingRecipe.length > 0) {
                    let mixR = 0, mixG = 0, mixB = 0;
                    item.mixingRecipe.forEach(part => {
                      mixR += part.paint.rgb[0] * part.parts;
                      mixG += part.paint.rgb[1] * part.parts;
                      mixB += part.paint.rgb[2] * part.parts;
                    });
                    mixR = Math.round(mixR / totalParts);
                    mixG = Math.round(mixG / totalParts);
                    mixB = Math.round(mixB / totalParts);
                    mixedHex = `#${mixR.toString(16).padStart(2, '0')}${mixG.toString(16).padStart(2, '0')}${mixB.toString(16).padStart(2, '0')}`.toUpperCase();
                  }

                  return (
                    <>
                      {/* Target vs Expected */}
                      <div className="modal-color-compare">
                        <div className="modal-color-block">
                          <div className="modal-swatch large" style={{ backgroundColor: item.dominantColor.hex }}>
                            <span className="modal-swatch-label">Target</span>
                          </div>
                          <div className="modal-color-info">
                            <span className="hex">{item.dominantColor.hex}</span>
                            <span className="rgb">RGB({item.dominantColor.r}, {item.dominantColor.g}, {item.dominantColor.b})</span>
                          </div>
                        </div>
                        <span className="modal-arrow">→</span>
                        <div className="modal-color-block">
                          <div className="modal-swatch large" style={{ backgroundColor: mixedHex }}>
                            <span className="modal-swatch-label">Result</span>
                          </div>
                          <div className="modal-color-info">
                            <span className="hex">{mixedHex}</span>
                          </div>
                        </div>
                      </div>

                      {/* Large Mix Bar */}
                      <div className="modal-mix-section">
                        <h3>Mixing Recipe</h3>
                        <div className="modal-mix-bar">
                          {item.mixingRecipe && item.mixingRecipe.map((part, partIdx) => {
                            const widthPercent = (part.parts / totalParts) * 100;
                            const r = part.paint.rgb[0];
                            const g = part.paint.rgb[1];
                            const b = part.paint.rgb[2];
                            const luminance = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
                            const textColor = luminance > 0.5 ? '#000000' : '#FFFFFF';
                            
                            return (
                              <div 
                                key={partIdx}
                                className="modal-mix-segment"
                                style={{ 
                                  width: `${widthPercent}%`,
                                  backgroundColor: part.paint.hex,
                                  color: textColor
                                }}
                              >
                                <span className="segment-name">{part.paint.name}</span>
                                <span className="segment-parts">{part.parts} part{part.parts > 1 ? 's' : ''}</span>
                              </div>
                            );
                          })}
                        </div>
                      </div>

                      {/* Paint Details */}
                      <div className="modal-paints-section">
                        <h3>Paints to Buy</h3>
                        <div className="modal-paint-list">
                          {item.mixingRecipe && item.mixingRecipe.map((part, partIdx) => (
                            <div key={partIdx} className="modal-paint-card">
                              <div 
                                className="modal-paint-swatch"
                                style={{ backgroundColor: part.paint.hex }}
                              />
                              <div className="modal-paint-info">
                                <div className="paint-name-large">{part.paint.name}</div>
                                <div className="paint-details-grid">
                                  <div className="detail-row">
                                    <span className="detail-label">Brand:</span>
                                    <span className="detail-value">{part.paint.brand}</span>
                                  </div>
                                  <div className="detail-row">
                                    <span className="detail-label">Series:</span>
                                    <span className="detail-value">{part.paint.series}</span>
                                  </div>
                                  <div className="detail-row">
                                    <span className="detail-label">Pigments:</span>
                                    <span className="detail-value pigment-code">{part.paint.pigments}</span>
                                  </div>
                                  <div className="detail-row">
                                    <span className="detail-label">Opacity:</span>
                                    <span className="detail-value">{part.paint.opacity}</span>
                                  </div>
                                  <div className="detail-row">
                                    <span className="detail-label">Hex:</span>
                                    <span className="detail-value">{part.paint.hex}</span>
                                  </div>
                                  <div className="detail-row">
                                    <span className="detail-label">Parts:</span>
                                    <span className="detail-value parts-value">{part.parts} of {totalParts}</span>
                                  </div>
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>

                      {/* Best Single Match */}
                      {item.paintMatches && item.paintMatches.length > 0 && (
                        <div className="modal-single-match">
                          <h3>Closest Single Paint</h3>
                          <div className="modal-paint-card">
                            <div 
                              className="modal-paint-swatch"
                              style={{ backgroundColor: item.paintMatches[0].paint.hex }}
                            />
                            <div className="modal-paint-info">
                              <div className="paint-name-large">{item.paintMatches[0].paint.name}</div>
                              <div className="paint-details-grid">
                                <div className="detail-row">
                                  <span className="detail-label">Match:</span>
                                  <span className="detail-value">{item.paintMatches[0].matchRating}</span>
                                </div>
                                <div className="detail-row">
                                  <span className="detail-label">ΔE:</span>
                                  <span className="detail-value">{item.paintMatches[0].deltaE.toFixed(2)}</span>
                                </div>
                                <div className="detail-row">
                                  <span className="detail-label">Pigments:</span>
                                  <span className="detail-value pigment-code">{item.paintMatches[0].paint.pigments}</span>
                                </div>
                              </div>
                            </div>
                          </div>
                        </div>
                      )}
                    </>
                  );
                })()}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
