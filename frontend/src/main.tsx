import React from 'react'
import ReactDOM from 'react-dom/client'
import { AppRouter } from './router.tsx'
import { Toaster } from "@/components/ui/sonner"
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AppRouter />
    <Toaster richColors position="top-center" />
  </React.StrictMode>,
)
