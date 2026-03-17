// PDF Handler 2: Paint-by-Numbers PDF
// Page 1: Overview (Paint-by-Numbers + Original side by side)
// Page 2: Full-page Paint-by-Numbers (minimal outlines, small numbers)
// Page 3: Shopping List
// Pages 4-N: One page per palette color with images + mixing recipe

import { LogInfo } from '../../wailsjs/runtime/runtime';
import { ExportPaintByNumbersPDF } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';
import type { PdfExportContext, PbnPaletteItem } from './types';
import {
  imageDataToBase64,
  getProcessedSourceImageData,
  buildShoppingList,
  generatePaintByNumbersImage,
  generatePaintByNumbersImageMinimal,
} from './utils';

export async function exportPdf2(ctx: PdfExportContext): Promise<void> {
  const { result, originalImageData, posterizedImageData, posterizeMode, smoothingPasses, tileSize } = ctx;
  
  if (!result || !originalImageData) {
    LogInfo('PDF Export 2: Missing required data');
    return;
  }
  
  // Get the processed source image (with posterize/smoothing/tiling applied)
  const sourceImageData = getProcessedSourceImageData(
    originalImageData,
    posterizedImageData,
    posterizeMode,
    smoothingPasses,
    tileSize
  );
  
  const usePosterized = posterizeMode && posterizedImageData !== null;
  
  // Generate paint-by-numbers outline image (no highlight for overview page)
  const palette = result.palette.map(p => p.dominantColor);
  const paintByNumbersImageData = generatePaintByNumbersImage(
    sourceImageData,
    palette,
    usePosterized,
    true, // showGrid
    -1   // no highlight
  );
  
  if (!paintByNumbersImageData) {
    LogInfo('PDF Export 2: Failed to generate paint-by-numbers image');
    return;
  }
  
  // Generate full-page minimal paint-by-numbers image
  const fullPagePbnImageData = generatePaintByNumbersImageMinimal(
    sourceImageData,
    palette,
    usePosterized
  );
  
  // Get original image as base64
  const originalImageBase64 = imageDataToBase64(originalImageData);
  if (!originalImageBase64) {
    LogInfo('PDF Export 2: Failed to generate original image data');
    return;
  }
  
  // Build shopping list
  const shoppingList = buildShoppingList(result);
  
  // Build palette data with highlighted paint-by-numbers images for each color
  const pbnPalette: PbnPaletteItem[] = result.palette.map((item, index) => {
    // Generate a highlighted version of the paint-by-numbers image for this color
    const highlightedPbnImage = generatePaintByNumbersImage(
      sourceImageData,
      palette,
      usePosterized,
      true,  // showGrid
      index  // highlight this color index
    );
    
    return {
      hex: item.dominantColor.hex,
      colorNumber: index + 1,
      highlightedPbnImage: highlightedPbnImage || '',
      mixingRecipe: item.mixingRecipe?.map(recipe => ({
        name: recipe.paint.name,
        parts: recipe.parts,
        hex: recipe.paint.hex,
      })) || [],
    };
  });
  
  try {
    const pdfData = main.PaintByNumbersPDFData.createFrom({
      paintByNumbersImageData: paintByNumbersImageData,
      fullPagePbnImageData: fullPagePbnImageData,
      originalImageData: originalImageBase64,
      shoppingList: shoppingList,
      palette: pbnPalette,
    });
    await ExportPaintByNumbersPDF(pdfData);
  } catch (err) {
    console.error('Failed to export Paint-by-Numbers PDF:', err);
  }
}

