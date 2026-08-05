import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.jsx'

// Bloqueio global da barra de Espaço: impede o navegador de rolar a página
// com a tecla, independente de onde estiver o foco. Preserva a digitação em
// campos de texto (ex.: nome do jogador).
window.addEventListener(
  'keydown',
  (e) => {
    const t = e.target;
    const typing =
      t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable);
    if ((e.code === 'Space' || e.key === ' ') && !typing) {
      e.preventDefault();
    }
  },
  { passive: false }
);

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
