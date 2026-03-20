import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom';
import AppLayout from './layouts/AppLayout';
import Dashboard from './pages/Dashboard';
import Login from './pages/Login';
import Status from './pages/Status';
import Subscriptions from './pages/Subscriptions';
import Interfaces from './pages/Interfaces';
import Listeners from './pages/Listeners';
import SystemConfig from './pages/SystemConfig';
import Events from './pages/Events';

const router = createBrowserRouter([
  {
    path: '/login',
    element: <Login />,
  },
  {
    path: '/',
    element: <AppLayout />,
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: 'dashboard', element: <Dashboard /> },
      { path: 'status', element: <Status /> },
      { path: 'subscriptions', element: <Subscriptions /> },
      { path: 'interfaces', element: <Interfaces /> },
      { path: 'listeners', element: <Listeners /> },
      { path: 'system', element: <SystemConfig /> },
      { path: 'events', element: <Events /> },
    ],
  },
]);

export function AppRouter() {
  return <RouterProvider router={router} />;
}
