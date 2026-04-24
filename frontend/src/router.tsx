import { Suspense, lazy, type ComponentType, type LazyExoticComponent } from 'react'
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import AppLayout from './layouts/AppLayout'

const Usage = lazy(() => import('./pages/Usage'))
const Routes = lazy(() => import('./pages/Routes'))
const Publications = lazy(() => import('./pages/Publications'))
const Runtime = lazy(() => import('./pages/Runtime'))
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
  { path: '/', element: <Navigate to="/usage" replace /> },
  { path: '/launch', element: renderLazyPage(Launch) },
  { path: '/login', element: renderLazyPage(Login) },
  { path: '/setup', element: renderLazyPage(Setup) },
  {
    element: <AppLayout />,
    children: [
      { path: '/usage', element: renderLazyPage(Usage) },
      { path: '/routes', element: renderLazyPage(Routes) },
      { path: '/publications', element: renderLazyPage(Publications) },
      { path: '/runtime', element: renderLazyPage(Runtime) },
      { path: '/system', element: renderLazyPage(SystemConfig) },
    ],
  },
  { path: '*', element: <Navigate to="/launch" replace /> },
])

export function AppRouter() {
  return <RouterProvider router={router} />
}
