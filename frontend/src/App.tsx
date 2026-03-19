import { useState, useEffect, useCallback, useRef } from 'react';
import { Routes, Route, useLocation, useNavigate } from 'react-router-dom';
import { IconPhoto, IconPalette, IconHeart } from '@tabler/icons-react';
import { WindowGetPosition, WindowGetSize, EventsOn, EventsOff } from '@wailsjs/runtime/runtime';
import { useWindowGeometry, AppLayout, SplashScreen, type NavItem } from '@trueblocks/ui';
import { DarkModeToggle } from '@trueblocks/ui';
import { ProjectsPage } from '@/pages/ProjectsPage';
import { PaintsPage } from '@/pages/PaintsPage';
import { FavoritesPage } from '@/pages/FavoritesPage';
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts';
import {
  GetSidebarWidth,
  SaveWindowGeometry,
  SaveLastRoute,
  GetLastRoute,
  GetTab,
  SetSidebarWidth,
} from '@app';
import { Log } from '@/utils';

const TABBED_PAGES = new Set(['projects', 'paints', 'favorites']);

function App() {
  const [showSplash, setShowSplash] = useState(true);
  const [initialSidebarWidth, setInitialSidebarWidth] = useState(220);
  const hasRestoredRoute = useRef(false);
  const location = useLocation();
  const navigate = useNavigate();
  useKeyboardShortcuts();
  useWindowGeometry(SaveWindowGeometry, WindowGetPosition, WindowGetSize);

  useEffect(() => {
    function handleUnhandledKey(e: KeyboardEvent) {
      if (e.metaKey || e.ctrlKey) return;
      const target = e.target as HTMLElement;
      const isInput =
        target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable;
      if (isInput) return;
      if (e.key === 'Tab') return;
      const arrowKeys = ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'];
      if (e.key.length === 1 || arrowKeys.includes(e.key)) {
        e.preventDefault();
      }
    }
    window.addEventListener('keydown', handleUnhandledKey);
    return () => window.removeEventListener('keydown', handleUnhandledKey);
  }, []);

  useEffect(() => {
    if (hasRestoredRoute.current) {
      SaveLastRoute(location.pathname);
    }
  }, [location.pathname]);

  useEffect(() => {
    if (hasRestoredRoute.current) return;
    hasRestoredRoute.current = true;

    Promise.all([GetLastRoute(), GetSidebarWidth()]).then(([lastRoute, sidebarWidth]) => {
      if (lastRoute && lastRoute !== '/' && lastRoute !== location.pathname) {
        navigate(lastRoute, { replace: true });
      }
      if (sidebarWidth && sidebarWidth > 0) {
        setInitialSidebarWidth(sidebarWidth);
      }
    });
  }, [location.pathname, navigate]);

  const navItems: NavItem[] = [
    { id: 'projects', label: 'Projects', icon: IconPhoto },
    { id: 'paints', label: 'Paints', icon: IconPalette },
    { id: 'favorites', label: 'Favorites', icon: IconHeart },
  ];

  const activeNav = location.pathname.split('/')[1] || 'projects';

  const handleNavigate = useCallback(
    async (id: string) => {
      const basePath = '/' + (location.pathname.split('/')[1] || '');
      const targetPath = '/' + id;

      if (basePath === targetPath && TABBED_PAGES.has(id)) {
        const currentTab = await GetTab(id);
        if (currentTab === 'detail' || !currentTab) {
          navigate(targetPath);
        }
      } else {
        navigate(targetPath);
      }
    },
    [location.pathname, navigate]
  );

  if (showSplash) {
    return (
      <SplashScreen
        title="Acrylic"
        subtitle="Paint Matching Studio"
        duration={2000}
        onComplete={() => setShowSplash(false)}
        onSubscribe={(handler) => {
          EventsOn('startup:status', handler);
          return () => EventsOff('startup:status');
        }}
      />
    );
  }

  Log('App rendered');

  return (
    <AppLayout
      title="Acrylic"
      subtitle="Paint Matching Studio"
      headerActions={<DarkModeToggle />}
      navItems={navItems}
      activeNav={activeNav}
      onNavigate={handleNavigate}
      initialSidebarWidth={initialSidebarWidth}
      saveSidebarWidth={SetSidebarWidth}
    >
      <Routes>
        <Route path="/" element={<ProjectsPage />} />
        <Route path="/projects" element={<ProjectsPage />} />
        <Route path="/projects/:id" element={<ProjectsPage />} />
        <Route path="/paints" element={<PaintsPage />} />
        <Route path="/paints/:id" element={<PaintsPage />} />
        <Route path="/favorites" element={<FavoritesPage />} />
        <Route path="/favorites/:id" element={<FavoritesPage />} />
      </Routes>
    </AppLayout>
  );
}

export default App;
