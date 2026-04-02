import { Suspense, lazy, type ComponentType, type LazyExoticComponent } from 'react';
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom';
import AppLayout from './layouts/AppLayout';

const Dashboard = lazy(() => import('./pages/Dashboard'));
const Login = lazy(() => import('./pages/Login'));
const Status = lazy(() => import('./pages/Status'));
const Subscriptions = lazy(() => import('./pages/Subscriptions'));
const Interfaces = lazy(() => import('./pages/Interfaces'));
const TransitRoutes = lazy(() => import('./pages/TransitRoutes'));
const Listeners = lazy(() => import('./pages/Listeners'));
const SystemConfig = lazy(() => import('./pages/SystemConfig'));
const Events = lazy(() => import('./pages/Events'));
const Setup = lazy(() => import('./pages/Setup'));

function RouteFallback() {
  return (
    <div className="flex min-h-[240px] items-center justify-center rounded-2xl border border-dashed border-zinc-200 bg-white/70 text-sm text-zinc-500">
      正在加载页面...
    </div>
  );
}

function renderLazyPage(Component: LazyExoticComponent<ComponentType>) {
  return (
    <Suspense fallback={<RouteFallback />}>
      <Component />
    </Suspense>
  );
}

const router = createBrowserRouter([
  {
    path: '/login',
    element: renderLazyPage(Login),
  },
  {
    path: '/',
    element: <AppLayout />,
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: 'dashboard', element: renderLazyPage(Dashboard) },
      { path: 'setup', element: renderLazyPage(Setup) },
      { path: 'status', element: renderLazyPage(Status) },
      { path: 'subscriptions', element: renderLazyPage(Subscriptions) },
      { path: 'interfaces', element: renderLazyPage(Interfaces) },
      { path: 'transits', element: renderLazyPage(TransitRoutes) },
      { path: 'listeners', element: renderLazyPage(Listeners) },
      { path: 'system', element: renderLazyPage(SystemConfig) },
      { path: 'events', element: renderLazyPage(Events) },
    ],
  },
]);

export function AppRouter() {
  return <RouterProvider router={router} />;
}
