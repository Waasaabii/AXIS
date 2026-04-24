import { Suspense, lazy, type ComponentType, type LazyExoticComponent } from 'react'
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import AppLayout from './layouts/AppLayout'
import DesktopShell from './layouts/DesktopShell'

const Home = lazy(() => import('./pages/Home'))
const Egress = lazy(() => import('./pages/Egress'))
const Rules = lazy(() => import('./pages/Rules'))
const Publications = lazy(() => import('./pages/Publications'))
const Diagnostics = lazy(() => import('./pages/Diagnostics'))
const SystemConfig = lazy(() => import('./pages/SystemConfig'))
const Launch = lazy(() => import('./pages/Launch'))
const Login = lazy(() => import('./pages/Login'))
const Setup = lazy(() => import('./pages/Setup'))

function RouteFallback() {
  return <div className="flex min-h-[240px] items-center justify-center rounded-2xl border border-dashed border-zinc-200 bg-white/70 text-sm text-zinc-500">正在加载页面...</div>
}

function renderLazyPage(Component: LazyExoticComponent<ComponentType>) {
  return <Suspense fallback={<RouteFallback />}><Component /></Suspense>
}

const router = createBrowserRouter([
  {
    element: <DesktopShell />,
    children: [
      { path: '/', element: <Navigate to="/home" replace /> },
      { path: '/launch', element: renderLazyPage(Launch) },
      { path: '/login', element: renderLazyPage(Login) },
      { path: '/setup', element: renderLazyPage(Setup) },
      {
        element: <AppLayout />,
        children: [
          { path: '/home', element: renderLazyPage(Home) },
          { path: '/egress', element: renderLazyPage(Egress) },
          { path: '/rules', element: renderLazyPage(Rules) },
          { path: '/publications', element: renderLazyPage(Publications) },
          { path: '/diagnostics', element: renderLazyPage(Diagnostics) },
          { path: '/system', element: renderLazyPage(SystemConfig) },
        ],
      },
      { path: '*', element: <Navigate to="/launch" replace /> },
    ],
  },
])

export function AppRouter() {
  return <RouterProvider router={router} />
}
