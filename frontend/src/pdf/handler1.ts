// PDF Handler 1: Full Comparison PDF
// Exports original + modified images, shopping list, and palette with isolation images

import { LogInfo } from '../../wailsjs/runtime/runtime';
import { ExportComparisonPDF } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';
import type { PdfExportContext } from './types';
import {
  canvasToBase64,
  imageDataToBase64,
  getProcessedSourceImageData,
  buildShoppingList,
  buildPaletteForPdf,
} from './utils';

export async function exportComparisonPdf(ctx: PdfExportContext): Promise<void> {
  const { result, originalImageData, posterizedImageData, posterizeMode, smoothingPasses, tileSize, canvasRef } = ctx;
  
  if (!result || !originalImageData || !canvasRef.current) {
    LogInfo('PDF Export 1: Missing required data');
    return;
  }
  
  const modifiedImageData = canvasToBase64(canvasRef.current);
  const originalImageBase64 = imageDataToBase64(originalImageData);
  
  if (!modifiedImageData || !originalImageBase64) {
    LogInfo('PDF Export 1: Failed to generate image data');
    return;
  }
  
  const sourceImageData = getProcessedSourceImageData(
    originalImageData,
    posterizedImageData,
    posterizeMode,
    smoothingPasses,
    tileSize
  );
  
  const usePosterized = posterizeMode && posterizedImageData !== null;
  const shoppingList = buildShoppingList(result);
  const palette = buildPaletteForPdf(result, sourceImageData, usePosterized);
  
  try {
    const pdfData = main.ComparisonPDFData.createFrom({
      modifiedImageData: modifiedImageData,
      originalImageData: originalImageBase64,
      shoppingList: shoppingList,
      palette: palette,
    });
    await ExportComparisonPDF(pdfData);
  } catch (err) {
    console.error('Failed to export PDF:', err);
  }
}
