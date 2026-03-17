// PDF Export Types

export interface ColorInfo {
  r: number;
  g: number;
  b: number;
  hex: string;
}

export interface PaintProduct {
  id: string;
  brand: string;
  name: string;
  series: number;
  opacity: string;
  pigments: string;
  rgb: number[];
  hex: string;
}

export interface MixingPart {
  paint: PaintProduct;
  parts: number;
}

export interface PaletteResult {
  dominantColor: ColorInfo;
  paintMatches: Array<{
    paint: PaintProduct;
    deltaE: number;
    matchRating: string;
  }>;
  mixingRecipe: MixingPart[];
}

export interface ProcessingResult {
  imageData: string;
  palette: PaletteResult[];
  imageWidth: number;
  imageHeight: number;
}

// Shopping list item for PDF
export interface ShoppingListItem {
  name: string;
  brand: string;
  series: number;
  pigments: string;
  opacity: string;
  hex: string;
}

// Palette item for PDF
export interface PdfPaletteItem {
  hex: string;
  colorNumber: number;
  mixingRecipe: Array<{
    name: string;
    parts: number;
    hex: string;
  }>;
  isolationImageData: string;
}

// Context passed to PDF handlers
export interface PdfExportContext {
  result: ProcessingResult | null;
  originalImageData: ImageData | null;
  posterizedImageData: ImageData | null;
  posterizeMode: boolean;
  smoothingPasses: number;
  tileSize: number;
  canvasRef: React.RefObject<HTMLCanvasElement>;
}

// Paint-by-Numbers palette item (with highlighted PBN image for each color)
export interface PbnPaletteItem {
  hex: string;
  colorNumber: number;
  highlightedPbnImage: string; // Paint-by-numbers image with this color's numbers highlighted
  mixingRecipe: Array<{
    name: string;
    parts: number;
    hex: string;
  }>;
}

// Data for Paint-by-Numbers PDF (Handler 2)
export interface PaintByNumbersPDFData {
  paintByNumbersImageData: string;  // Base64 PNG of paint-by-numbers outline
  fullPagePbnImageData: string;     // Base64 PNG of minimal full-page paint-by-numbers
  originalImageData: string;         // Base64 PNG of original image
  shoppingList: ShoppingListItem[];  // Unique paints to buy
  palette: PbnPaletteItem[];         // Color palette with mixing recipes
}

