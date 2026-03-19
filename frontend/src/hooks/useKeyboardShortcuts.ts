import { useEffect } from 'react';
import {
  ExportComparisonPDF,
  ExportPaintByNumbersPDF,
  ExportColorDetailPDF,
  ExportShoppingListPDF,
} from '@app';
import { LogErr } from '@/utils';
import { useLocation } from 'react-router-dom';

export function useKeyboardShortcuts() {
  const location = useLocation();

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (!e.metaKey) return;

      const path = location.pathname;
      const isProjectDetail = /^\/projects\/\d+$/.test(path);
      const projectId = isProjectDetail ? parseInt(path.split('/')[2], 10) : null;

      if (projectId !== null) {
        if (e.key === '1') {
          e.preventDefault();
          ExportComparisonPDF(projectId).catch((err) =>
            LogErr('Export comparison PDF failed:', err)
          );
        } else if (e.key === '2') {
          e.preventDefault();
          ExportPaintByNumbersPDF(projectId).catch((err) =>
            LogErr('Export paint-by-numbers PDF failed:', err)
          );
        } else if (e.key === '3') {
          e.preventDefault();
          ExportColorDetailPDF(projectId, 0).catch((err) =>
            LogErr('Export color detail PDF failed:', err)
          );
        } else if (e.key === '4') {
          e.preventDefault();
          ExportShoppingListPDF(projectId).catch((err) =>
            LogErr('Export acrylic list PDF failed:', err)
          );
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [location.pathname]);
}
