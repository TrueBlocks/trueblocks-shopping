export namespace color {
	
	export class Lab {
	    L: number;
	    A: number;
	    B: number;
	
	    static createFrom(source: any = {}) {
	        return new Lab(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.L = source["L"];
	        this.A = source["A"];
	        this.B = source["B"];
	    }
	}

}

export namespace inventory {
	
	export class PaintProduct {
	    id: string;
	    brand: string;
	    name: string;
	    series: number;
	    opacity: string;
	    pigments: string;
	    rgb: number[];
	    hex: string;
	    lab: color.Lab;
	
	    static createFrom(source: any = {}) {
	        return new PaintProduct(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.brand = source["brand"];
	        this.name = source["name"];
	        this.series = source["series"];
	        this.opacity = source["opacity"];
	        this.pigments = source["pigments"];
	        this.rgb = source["rgb"];
	        this.hex = source["hex"];
	        this.lab = this.convertValues(source["lab"], color.Lab);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class ColorInfo {
	    r: number;
	    g: number;
	    b: number;
	    hex: string;
	
	    static createFrom(source: any = {}) {
	        return new ColorInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.r = source["r"];
	        this.g = source["g"];
	        this.b = source["b"];
	        this.hex = source["hex"];
	    }
	}
	export class MixingRecipeItem {
	    name: string;
	    parts: number;
	    hex: string;
	
	    static createFrom(source: any = {}) {
	        return new MixingRecipeItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.parts = source["parts"];
	        this.hex = source["hex"];
	    }
	}
	export class PaletteColorInfo {
	    hex: string;
	    colorNumber: number;
	    mixingRecipe: MixingRecipeItem[];
	    isolationImageData: string;
	
	    static createFrom(source: any = {}) {
	        return new PaletteColorInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hex = source["hex"];
	        this.colorNumber = source["colorNumber"];
	        this.mixingRecipe = this.convertValues(source["mixingRecipe"], MixingRecipeItem);
	        this.isolationImageData = source["isolationImageData"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ShoppingListPaint {
	    name: string;
	    brand: string;
	    series: number;
	    pigments: string;
	    opacity: string;
	    hex: string;
	
	    static createFrom(source: any = {}) {
	        return new ShoppingListPaint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.brand = source["brand"];
	        this.series = source["series"];
	        this.pigments = source["pigments"];
	        this.opacity = source["opacity"];
	        this.hex = source["hex"];
	    }
	}
	export class ComparisonPDFData {
	    modifiedImageData: string;
	    originalImageData: string;
	    shoppingList: ShoppingListPaint[];
	    palette: PaletteColorInfo[];
	
	    static createFrom(source: any = {}) {
	        return new ComparisonPDFData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modifiedImageData = source["modifiedImageData"];
	        this.originalImageData = source["originalImageData"];
	        this.shoppingList = this.convertValues(source["shoppingList"], ShoppingListPaint);
	        this.palette = this.convertValues(source["palette"], PaletteColorInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MixingPart {
	    paint: inventory.PaintProduct;
	    parts: number;
	
	    static createFrom(source: any = {}) {
	        return new MixingPart(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paint = this.convertValues(source["paint"], inventory.PaintProduct);
	        this.parts = source["parts"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PDFPaintPart {
	    name: string;
	    brand: string;
	    series: number;
	    pigments: string;
	    opacity: string;
	    hex: string;
	    rgb: number[];
	    parts: number;
	
	    static createFrom(source: any = {}) {
	        return new PDFPaintPart(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.brand = source["brand"];
	        this.series = source["series"];
	        this.pigments = source["pigments"];
	        this.opacity = source["opacity"];
	        this.hex = source["hex"];
	        this.rgb = source["rgb"];
	        this.parts = source["parts"];
	    }
	}
	export class PDFExportData {
	    colorIndex: number;
	    imageData: string;
	    originalImageData: string;
	    targetHex: string;
	    targetRGB: number[];
	    resultHex: string;
	    mixingRecipe: PDFPaintPart[];
	    totalParts: number;
	
	    static createFrom(source: any = {}) {
	        return new PDFExportData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.colorIndex = source["colorIndex"];
	        this.imageData = source["imageData"];
	        this.originalImageData = source["originalImageData"];
	        this.targetHex = source["targetHex"];
	        this.targetRGB = source["targetRGB"];
	        this.resultHex = source["resultHex"];
	        this.mixingRecipe = this.convertValues(source["mixingRecipe"], PDFPaintPart);
	        this.totalParts = source["totalParts"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PbnPaletteItem {
	    hex: string;
	    colorNumber: number;
	    highlightedPbnImage: string;
	    mixingRecipe: MixingRecipeItem[];
	
	    static createFrom(source: any = {}) {
	        return new PbnPaletteItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hex = source["hex"];
	        this.colorNumber = source["colorNumber"];
	        this.highlightedPbnImage = source["highlightedPbnImage"];
	        this.mixingRecipe = this.convertValues(source["mixingRecipe"], MixingRecipeItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PaintByNumbersPDFData {
	    paintByNumbersImageData: string;
	    fullPagePbnImageData: string;
	    originalImageData: string;
	    shoppingList: ShoppingListPaint[];
	    palette: PbnPaletteItem[];
	
	    static createFrom(source: any = {}) {
	        return new PaintByNumbersPDFData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paintByNumbersImageData = source["paintByNumbersImageData"];
	        this.fullPagePbnImageData = source["fullPagePbnImageData"];
	        this.originalImageData = source["originalImageData"];
	        this.shoppingList = this.convertValues(source["shoppingList"], ShoppingListPaint);
	        this.palette = this.convertValues(source["palette"], PbnPaletteItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PaintMatch {
	    paint: inventory.PaintProduct;
	    deltaE: number;
	    matchRating: string;
	
	    static createFrom(source: any = {}) {
	        return new PaintMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paint = this.convertValues(source["paint"], inventory.PaintProduct);
	        this.deltaE = source["deltaE"];
	        this.matchRating = source["matchRating"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class PaletteResult {
	    dominantColor: ColorInfo;
	    paintMatches: PaintMatch[];
	    mixingRecipe: MixingPart[];
	
	    static createFrom(source: any = {}) {
	        return new PaletteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dominantColor = this.convertValues(source["dominantColor"], ColorInfo);
	        this.paintMatches = this.convertValues(source["paintMatches"], PaintMatch);
	        this.mixingRecipe = this.convertValues(source["mixingRecipe"], MixingPart);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ProcessingResult {
	    imageData: string;
	    palette: PaletteResult[];
	    imageWidth: number;
	    imageHeight: number;
	
	    static createFrom(source: any = {}) {
	        return new ProcessingResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.imageData = source["imageData"];
	        this.palette = this.convertValues(source["palette"], PaletteResult);
	        this.imageWidth = source["imageWidth"];
	        this.imageHeight = source["imageHeight"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RecentImageInfo {
	    originalPath: string;
	    copiedPath: string;
	    filename: string;
	    processedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new RecentImageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.originalPath = source["originalPath"];
	        this.copiedPath = source["copiedPath"];
	        this.filename = source["filename"];
	        this.processedAt = source["processedAt"];
	    }
	}
	
	export class WindowSettings {
	    x: number;
	    y: number;
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new WindowSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}

}

