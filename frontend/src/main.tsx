import React from 'react'
import ReactDOM from 'react-dom/client'
import './i18n'
import './styles.css'
import { App } from './App'
import { ErrorBoundary } from './components/error-boundary'
import { applyTheme, preferredDark } from './lib/theme'

applyTheme(preferredDark())

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </React.StrictMode>,
)
